package eduvpn

import (
	"errors"
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

type (
	FSMStateID    = fsm.FSMStateID
	FSMStates     = fsm.FSMStates
	FSMState      = fsm.FSMState
	FSMTransition = fsm.FSMTransition
)

const (
	// Deregistered means the app is not registered with the wrapper
	STATE_DEREGISTERED FSMStateID = iota

	// No Server means the user has not chosen a server yet
	STATE_NO_SERVER

	// The user selected a Secure Internet server but needs to choose a location
	STATE_ASK_LOCATION

	// The user is currently selecting a server in the UI
	STATE_SEARCH_SERVER

	// We are loading the server details
	STATE_LOADING_SERVER

	// Chosen Server means the user has chosen a server to connect to
	STATE_CHOSEN_SERVER

	// OAuth Started means the OAuth process has started
	STATE_OAUTH_STARTED

	// Authorized means the OAuth process has finished and the user is now authorized with the server
	STATE_AUTHORIZED

	// Requested config means the user has requested a config for connecting
	STATE_REQUEST_CONFIG

	// Ask profile means the go code is asking for a profile selection from the UI
	STATE_ASK_PROFILE

	// Disconnected means the user has gotten a config for a server but is not connected yet
	STATE_DISCONNECTED

	// Disconnecting means the OS is disconnecting and the Go code is doing the /disconnect
	STATE_DISCONNECTING

	// Connecting means the OS is establishing a connection to the server
	STATE_CONNECTING

	// Connected means the user has been connected to the server
	STATE_CONNECTED
)

func GetStateName(s FSMStateID) string {
	switch s {
	case STATE_DEREGISTERED:
		return "Deregistered"
	case STATE_NO_SERVER:
		return "No_Server"
	case STATE_ASK_LOCATION:
		return "Ask_Location"
	case STATE_SEARCH_SERVER:
		return "Search_Server"
	case STATE_LOADING_SERVER:
		return "Loading_Server"
	case STATE_CHOSEN_SERVER:
		return "Chosen_Server"
	case STATE_OAUTH_STARTED:
		return "OAuth_Started"
	case STATE_DISCONNECTED:
		return "Disconnected"
	case STATE_REQUEST_CONFIG:
		return "Request_Config"
	case STATE_ASK_PROFILE:
		return "Ask_Profile"
	case STATE_AUTHORIZED:
		return "Authorized"
	case STATE_DISCONNECTING:
		return "Disconnecting"
	case STATE_CONNECTING:
		return "Connecting"
	case STATE_CONNECTED:
		return "Connected"
	default:
		panic("unknown conversion of state to string")
	}
}

func newFSM(
	name string,
	callback func(FSMStateID, FSMStateID, interface{}),
	directory string,
	debug bool,
) fsm.FSM {
	states := FSMStates{
		STATE_DEREGISTERED: FSMState{
			Transitions: []FSMTransition{{To: STATE_NO_SERVER, Description: "Client registers"}},
		},
		STATE_NO_SERVER: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_NO_SERVER, Description: "Reload list"},
				{To: STATE_CHOSEN_SERVER, Description: "User chooses a server"},
				{To: STATE_SEARCH_SERVER, Description: "The user is trying to choose a Server in the UI"},
				{To: STATE_CONNECTED, Description: "The user is already connected"},
				{To: STATE_ASK_LOCATION, Description: "Change the location in the main screen"},
			},
		},
		STATE_SEARCH_SERVER: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_LOADING_SERVER, Description: "User clicks a server in the UI"},
				{To: STATE_NO_SERVER, Description: "Cancel or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_ASK_LOCATION: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_CHOSEN_SERVER, Description: "Location chosen"},
				{To: STATE_NO_SERVER, Description: "Go back or Error"},
				{To: STATE_SEARCH_SERVER, Description: "Cancel or Error"},
			},
		},
		STATE_LOADING_SERVER: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_CHOSEN_SERVER, Description: "Server info loaded"},
				{
					To: STATE_ASK_LOCATION,
					Description: "User chooses a Secure Internet server but no location is configured",
				},
				{To: STATE_NO_SERVER, Description: "Go back or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_CHOSEN_SERVER: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_AUTHORIZED, Description: "Found tokens in config"},
				{To: STATE_OAUTH_STARTED, Description: "No tokens found in config"},
			},
		},
		STATE_OAUTH_STARTED: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_AUTHORIZED, Description: "User authorizes with browser"},
				{To: STATE_NO_SERVER, Description: "Cancel or Error"},
				{To: STATE_SEARCH_SERVER, Description: "Cancel or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_AUTHORIZED: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_OAUTH_STARTED, Description: "Re-authorize with OAuth"},
				{To: STATE_REQUEST_CONFIG, Description: "Client requests a config"},
			},
		},
		STATE_REQUEST_CONFIG: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_ASK_PROFILE, Description: "Multiple profiles found and no profile chosen"},
				{To: STATE_DISCONNECTED, Description: "Only one profile or profile already chosen"},
				{To: STATE_NO_SERVER, Description: "Cancel or Error"},
				{To: STATE_OAUTH_STARTED, Description: "Re-authorize"},
			},
		},
		STATE_ASK_PROFILE: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_DISCONNECTED, Description: "User chooses profile"},
				{To: STATE_NO_SERVER, Description: "Cancel or Error"},
				{To: STATE_SEARCH_SERVER, Description: "Cancel or Error"},
			},
		},
		STATE_DISCONNECTED: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_CONNECTING, Description: "OS reports it is trying to connect"},
				{To: STATE_REQUEST_CONFIG, Description: "User reconnects"},
				{To: STATE_NO_SERVER, Description: "User wants to choose a new server"},
				{To: STATE_OAUTH_STARTED, Description: "Re-authorize with OAuth"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_DISCONNECTING: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_DISCONNECTED, Description: "Cancel or Error"},
				{To: STATE_DISCONNECTED, Description: "Done disconnecting"},
			},
		},
		STATE_CONNECTING: FSMState{
			Transitions: []FSMTransition{
				{To: STATE_DISCONNECTED, Description: "Cancel or Error"},
				{To: STATE_CONNECTED, Description: "Done connecting"},
			},
		},
		STATE_CONNECTED: FSMState{
			Transitions: []FSMTransition{{To: STATE_DISCONNECTING, Description: "App wants to disconnect"}},
		},
	}
	returnedFSM := fsm.FSM{}
	returnedFSM.Init(name, STATE_DEREGISTERED, states, callback, directory, GetStateName, debug)
	return returnedFSM
}

type FSMDeregisteredError struct{}

func (e FSMDeregisteredError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{
		Message: "Client not registered with the GO library",
		Err: errors.New(
			"the current FSM state is deregistered, but the function needs a state that is not deregistered",
		),
	}
}

type FSMWrongStateTransitionError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e FSMWrongStateTransitionError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{
		Message: "Wrong FSM transition",
		Err: fmt.Errorf(
			"wrong FSM state, got: %s, want: a state with a transition to: %s",
			GetStateName(e.Got),
			GetStateName(e.Want),
		),
	}
}

type FSMWrongStateError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e FSMWrongStateError) CustomError() *types.WrappedErrorMessage {
	return &types.WrappedErrorMessage{
		Message: "Wrong FSM State",
		Err: fmt.Errorf(
			"wrong FSM state, got: %s, want: %s",
			GetStateName(e.Got),
			GetStateName(e.Want),
		),
	}
}
