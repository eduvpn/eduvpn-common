package fsm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"

	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

type (
	FSMStateID      int8
	FSMStateIDSlice []FSMStateID
)

func (v FSMStateIDSlice) Len() int {
	return len(v)
}

func (v FSMStateIDSlice) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v FSMStateIDSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

const (
	// Deregistered means the app is not registered with the wrapper
	DEREGISTERED FSMStateID = iota

	// No Server means the user has not chosen a server yet
	NO_SERVER

	// The user selected a Secure Internet server but needs to choose a location
	ASK_LOCATION

	// The user is currently selecting a server in the UI
	SEARCH_SERVER

	// We are loading the server details
	LOADING_SERVER

	// Chosen Server means the user has chosen a server to connect to
	CHOSEN_SERVER

	// OAuth Started means the OAuth process has started
	OAUTH_STARTED

	// Authorized means the OAuth process has finished and the user is now authorized with the server
	AUTHORIZED

	// Requested config means the user has requested a config for connecting
	REQUEST_CONFIG

	// Ask profile means the go code is asking for a profile selection from the UI
	ASK_PROFILE

	// Has config means the user has gotten a config
	HAS_CONFIG

	// Connecting means the OS is establishing a connection to the server
	CONNECTING

	// Connected means the user has been connected to the server
	CONNECTED
)

func (s FSMStateID) String() string {
	switch s {
	case DEREGISTERED:
		return "Deregistered"
	case NO_SERVER:
		return "No_Server"
	case ASK_LOCATION:
		return "Ask_Location"
	case SEARCH_SERVER:
		return "Search_Server"
	case LOADING_SERVER:
		return "Loading_Server"
	case CHOSEN_SERVER:
		return "Chosen_Server"
	case OAUTH_STARTED:
		return "OAuth_Started"
	case HAS_CONFIG:
		return "Has_Config"
	case REQUEST_CONFIG:
		return "Request_Config"
	case ASK_PROFILE:
		return "Ask_Profile"
	case AUTHORIZED:
		return "Authorized"
	case CONNECTING:
		return "Connecting"
	case CONNECTED:
		return "Connected"
	default:
		panic("unknown conversion of state to string")
	}
}

type FSMTransition struct {
	To          FSMStateID
	Description string
}

type (
	FSMStates map[FSMStateID]FSMState
)

type FSMState struct {
	Transitions []FSMTransition

	// Which state to go back to on a back transition
	BackState FSMStateID
}

type FSM struct {
	States  FSMStates
	Current FSMStateID

	// Info to be passed from the parent state
	Name          string
	StateCallback func(string, string, string)
	Directory     string
	Debug         bool
}

func (fsm *FSM) Init(name string, callback func(string, string, string), directory string, debug bool) {
	fsm.States = FSMStates{
		DEREGISTERED:   FSMState{Transitions: []FSMTransition{{NO_SERVER, "Client registers"}}},
		NO_SERVER:      FSMState{Transitions: []FSMTransition{{CHOSEN_SERVER, "User chooses a server"}, {SEARCH_SERVER, "The user is trying to choose a Server in the UI"}, {CONNECTED, "The user is already connected"}, {ASK_LOCATION, "Change the location in the main screen"}}},
		SEARCH_SERVER:  FSMState{Transitions: []FSMTransition{{LOADING_SERVER, "User clicks a server in the UI"}, {NO_SERVER, "Cancel or Error"}}, BackState: NO_SERVER},
		ASK_LOCATION:   FSMState{Transitions: []FSMTransition{{CHOSEN_SERVER, "Location chosen"}, {NO_SERVER, "Go back or Error"}, {SEARCH_SERVER, "Cancel or Error"}}},
		LOADING_SERVER: FSMState{Transitions: []FSMTransition{{CHOSEN_SERVER, "Server info loaded"}, {ASK_LOCATION, "User chooses a Secure Internet server but no location is configured"}}},
		CHOSEN_SERVER:  FSMState{Transitions: []FSMTransition{{AUTHORIZED, "Found tokens in config"}, {OAUTH_STARTED, "No tokens found in config"}}},
		OAUTH_STARTED:  FSMState{Transitions: []FSMTransition{{AUTHORIZED, "User authorizes with browser"}, {NO_SERVER, "Cancel or Error"}, {SEARCH_SERVER, "Cancel or Error"}}, BackState: NO_SERVER},
		AUTHORIZED:     FSMState{Transitions: []FSMTransition{{OAUTH_STARTED, "Re-authorize with OAuth"}, {REQUEST_CONFIG, "Client requests a config"}}},
		REQUEST_CONFIG: FSMState{Transitions: []FSMTransition{{ASK_PROFILE, "Multiple profiles found and no profile chosen"}, {HAS_CONFIG, "Only one profile or profile already chosen"}, {NO_SERVER, "Cancel or Error"}, {OAUTH_STARTED, "Re-authorize"}}},
		ASK_PROFILE:    FSMState{Transitions: []FSMTransition{{HAS_CONFIG, "User chooses profile"}, {NO_SERVER, "Cancel or Error"}, {SEARCH_SERVER, "Cancel or Error"}}},
		HAS_CONFIG:     FSMState{Transitions: []FSMTransition{{CONNECTING, "OS reports it is trying to connect"}, {REQUEST_CONFIG, "User chooses a new profile"}, {NO_SERVER, "User wants to choose a new server"}}, BackState: NO_SERVER},
		CONNECTING:     FSMState{Transitions: []FSMTransition{{HAS_CONFIG, "Cancel or Error"}, {CONNECTED, "Done connecting"}}},
		CONNECTED:      FSMState{Transitions: []FSMTransition{{HAS_CONFIG, "OS reports disconnected"}}},
	}
	fsm.Current = DEREGISTERED
	fsm.Name = name
	fsm.StateCallback = callback
	fsm.Directory = directory
	fsm.Debug = debug
}

func (fsm *FSM) InState(check FSMStateID) bool {
	return check == fsm.Current
}

func (fsm *FSM) HasTransition(check FSMStateID) bool {
	for _, transition_state := range fsm.States[fsm.Current].Transitions {
		if transition_state.To == check {
			return true
		}
	}

	return false
}

func (fsm *FSM) getGraphFilename(extension string) string {
	debugPath := path.Join(fsm.Directory, fsm.Name)
	return fmt.Sprintf("%s%s", debugPath, extension)
}

func (fsm *FSM) writeGraph() {
	graph := fsm.GenerateGraph()
	graphFile := fsm.getGraphFilename(".graph")
	graphImgFile := fsm.getGraphFilename(".png")
	f, err := os.Create(graphFile)
	if err != nil {
		return
	}

	f.WriteString(graph)
	f.Close()
	cmd := exec.Command("mmdc", "-i", graphFile, "-o", graphImgFile, "--scale", "4")

	cmd.Start()
}

func (fsm *FSM) GoBack() {
	fsm.GoTransition(fsm.States[fsm.Current].BackState)
}

func (fsm *FSM) GoTransitionWithData(newState FSMStateID, data string, background bool) bool {
	ok := fsm.HasTransition(newState)

	if ok {
		oldState := fsm.Current
		fsm.Current = newState
		if fsm.Debug {
			fsm.writeGraph()
		}

		if background {
			go fsm.StateCallback(oldState.String(), newState.String(), data)
		} else {
			fsm.StateCallback(oldState.String(), newState.String(), data)
		}
	}

	return ok
}

func (fsm *FSM) GoTransition(newState FSMStateID) bool {
	return fsm.GoTransitionWithData(newState, "{}", false)
}

func (fsm *FSM) generateMermaidGraph() string {
	graph := "graph TD\n"
	sorted_fsm := make(FSMStateIDSlice, 0, len(fsm.States))
	for state_id := range fsm.States {
		sorted_fsm = append(sorted_fsm, state_id)
	}
	sort.Sort(sorted_fsm)
	for _, state := range sorted_fsm {
		transitions := fsm.States[state].Transitions
		for _, transition := range transitions {
			if state == fsm.Current {
				graph += "\nstyle " + state.String() + " fill:cyan\n"
			} else {
				graph += "\nstyle " + state.String() + " fill:white\n"
			}
			graph += state.String() + "(" + state.String() + ") " + "-->|" + transition.Description + "| " + transition.To.String() + "\n"
		}
	}
	return graph
}

func (fsm *FSM) GenerateGraph() string {
	return fsm.generateMermaidGraph()
}

type DeregisteredError struct{}

func (e DeregisteredError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{Message: "Client not registered with the GO library", Err: errors.New("the current FSM state is deregistered, but the function needs a state that is not deregistered")}
}

type WrongStateTransitionError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e WrongStateTransitionError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{Message: "Wrong FSM transition", Err: errors.New(fmt.Sprintf("wrong FSM state, got: %s, want: a state with a transition to: %s", e.Got.String(), e.Want.String()))}
}

type WrongStateError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e WrongStateError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{Message: "Wrong FSM State", Err: errors.New(fmt.Sprintf("wrong FSM state, got: %s, want: %s", e.Got.String(), e.Want.String()))}
}
