package internal

import (
	"fmt"
	"os"
	"sort"
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

	// Chosen Server means the user has chosen a server to connect to
	CHOSEN_SERVER

	// OAuth Started means the OAuth process has started
	OAUTH_STARTED

	// Authenticated means the OAuth process has finished and the user is now authenticated with the server
	AUTHENTICATED

	// Requested config means the user has requested a config for connecting
	REQUEST_CONFIG

	// Has config means the user has gotten a config
	HAS_CONFIG

	// Ask profile means the go code is asking for a profile selection from the ui
	ASK_PROFILE

	// Connected means the user has been connected to the server
	CONNECTED
)

func (s FSMStateID) String() string {
	switch s {
	case DEREGISTERED:
		return "Deregistered"
	case NO_SERVER:
		return "No_Server"
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
	case AUTHENTICATED:
		return "Authenticated"
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
	FSMTransitions []FSMTransition
	FSMStates      map[FSMStateID]FSMTransitions
)

type FSM struct {
	States  FSMStates
	Current FSMStateID

	// Info to be passed from the parent state
	StateCallback func(string, string, string)
	Logger        *FileLogger
	Debug         bool
}

func (fsm *FSM) Init(callback func(string, string, string), logger *FileLogger, debug bool) {
	fsm.States = FSMStates{
		DEREGISTERED:   {{NO_SERVER, "Client registers"}},
		NO_SERVER:      {{CHOSEN_SERVER, "User chooses a server"}},
		CHOSEN_SERVER:  {{AUTHENTICATED, "Found tokens in config"}, {OAUTH_STARTED, "No tokens found in config"}},
		OAUTH_STARTED:  {{AUTHENTICATED, "User authorizes with browser"}},
		AUTHENTICATED:  {{OAUTH_STARTED, "Re-authenticate with OAuth"}, {REQUEST_CONFIG, "Client requests a config"}},
		REQUEST_CONFIG: {{ASK_PROFILE, "Multiple profiles found"}, {HAS_CONFIG, "Success, only one profile"}},
		ASK_PROFILE:    {{HAS_CONFIG, "User chooses profile and success"}},
		HAS_CONFIG:     {{CONNECTED, "OS reports connected"}},
		CONNECTED:      {{AUTHENTICATED, "OS reports disconnected"}},
	}
	fsm.Current = DEREGISTERED
	fsm.StateCallback = callback
	fsm.Logger = logger
	fsm.Debug = debug
}

func (fsm *FSM) InState(check FSMStateID) bool {
	return check == fsm.Current
}

func (fsm *FSM) HasTransition(check FSMStateID) bool {
	for _, transition_state := range fsm.States[fsm.Current] {
		if transition_state.To == check {
			return true
		}
	}

	return false
}

func (fsm *FSM) writeGraph() {
	graph := fsm.GenerateGraph()

	f, err := os.Create("debug.graph")
	if err != nil {
		fsm.Logger.Log(LOG_INFO, fmt.Sprintf("Failed to write debug fsm graph with error %v", err))
	}

	defer f.Close()

	f.WriteString(graph)
}

func (fsm *FSM) GoTransitionWithData(newState FSMStateID, data string, background bool) bool {
	ok := fsm.HasTransition(newState)

	if ok {
		oldState := fsm.Current
		fsm.Current = newState
		if fsm.Debug {
			fsm.writeGraph()
		}

		fsm.Logger.Log(LOG_INFO, fmt.Sprintf("State: %s -> State: %s with data %s\n", oldState.String(), newState.String(), data))

		if background {
			go fsm.StateCallback(oldState.String(), newState.String(), data)
		} else {
			fsm.StateCallback(oldState.String(), newState.String(), data)
		}
	}

	return ok
}

func (fsm *FSM) GoTransition(newState FSMStateID) bool {
	return fsm.GoTransitionWithData(newState, "", false)
}

func (fsm *FSM) generateMermaidGraph() string {
	graph := "graph TD\n"
	sorted_fsm := make(FSMStateIDSlice, 0, len(fsm.States))
	for state_id := range fsm.States {
		sorted_fsm = append(sorted_fsm, state_id)
	}
	sort.Sort(sorted_fsm)
	for _, state := range sorted_fsm {
		transitions := fsm.States[state]
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
