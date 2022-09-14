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
			Transitions: []FSMTransition{{STATE_NO_SERVER, "Client registers"}},
		},
		STATE_NO_SERVER: FSMState{
			Transitions: []FSMTransition{
				{STATE_NO_SERVER, "Reload list"},
				{STATE_CHOSEN_SERVER, "User chooses a server"},
				{STATE_SEARCH_SERVER, "The user is trying to choose a Server in the UI"},
				{STATE_CONNECTED, "The user is already connected"},
				{STATE_ASK_LOCATION, "Change the location in the main screen"},
			},
		},
		STATE_SEARCH_SERVER: FSMState{
			Transitions: []FSMTransition{
				{STATE_LOADING_SERVER, "User clicks a server in the UI"},
				{STATE_NO_SERVER, "Cancel or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_ASK_LOCATION: FSMState{
			Transitions: []FSMTransition{
				{STATE_CHOSEN_SERVER, "Location chosen"},
				{STATE_NO_SERVER, "Go back or Error"},
				{STATE_SEARCH_SERVER, "Cancel or Error"},
			},
		},
		STATE_LOADING_SERVER: FSMState{
			Transitions: []FSMTransition{
				{STATE_CHOSEN_SERVER, "Server info loaded"},
				{
					STATE_ASK_LOCATION,
					"User chooses a Secure Internet server but no location is configured",
				},
				{STATE_NO_SERVER, "Go back or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_CHOSEN_SERVER: FSMState{
			Transitions: []FSMTransition{
				{STATE_AUTHORIZED, "Found tokens in config"},
				{STATE_OAUTH_STARTED, "No tokens found in config"},
			},
		},
		STATE_OAUTH_STARTED: FSMState{
			Transitions: []FSMTransition{
				{STATE_AUTHORIZED, "User authorizes with browser"},
				{STATE_NO_SERVER, "Cancel or Error"},
				{STATE_SEARCH_SERVER, "Cancel or Error"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_AUTHORIZED: FSMState{
			Transitions: []FSMTransition{
				{STATE_OAUTH_STARTED, "Re-authorize with OAuth"},
				{STATE_REQUEST_CONFIG, "Client requests a config"},
			},
		},
		STATE_REQUEST_CONFIG: FSMState{
			Transitions: []FSMTransition{
				{STATE_ASK_PROFILE, "Multiple profiles found and no profile chosen"},
				{STATE_DISCONNECTED, "Only one profile or profile already chosen"},
				{STATE_NO_SERVER, "Cancel or Error"},
				{STATE_OAUTH_STARTED, "Re-authorize"},
			},
		},
		STATE_ASK_PROFILE: FSMState{
			Transitions: []FSMTransition{
				{STATE_DISCONNECTED, "User chooses profile"},
				{STATE_NO_SERVER, "Cancel or Error"},
				{STATE_SEARCH_SERVER, "Cancel or Error"},
			},
		},
		STATE_DISCONNECTED: FSMState{
			Transitions: []FSMTransition{
				{STATE_CONNECTING, "OS reports it is trying to connect"},
				{STATE_REQUEST_CONFIG, "User reconnects"},
				{STATE_NO_SERVER, "User wants to choose a new server"},
				{STATE_OAUTH_STARTED, "Re-authorize with OAuth"},
			},
			BackState: STATE_NO_SERVER,
		},
		STATE_DISCONNECTING: FSMState{
			Transitions: []FSMTransition{
				{STATE_DISCONNECTED, "Cancel or Error"},
				{STATE_DISCONNECTED, "Done disconnecting"},
			},
		},
		STATE_CONNECTING: FSMState{
			Transitions: []FSMTransition{
				{STATE_DISCONNECTED, "Cancel or Error"},
				{STATE_CONNECTED, "Done connecting"},
			},
		},
		STATE_CONNECTED: FSMState{
			Transitions: []FSMTransition{{STATE_DISCONNECTING, "App wants to disconnect"}},
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
		Err: errors.New(
			fmt.Sprintf(
				"wrong FSM state, got: %s, want: a state with a transition to: %s",
				GetStateName(e.Got),
				GetStateName(e.Want),
			),
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
		Err: errors.New(
			fmt.Sprintf(
				"wrong FSM state, got: %s, want: %s",
				GetStateName(e.Got),
				GetStateName(e.Want),
			),
		),
	}
}
