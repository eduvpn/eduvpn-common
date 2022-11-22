package client

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/types"
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
	callback func(FSMStateID, FSMStateID, interface{}) bool,
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
				{To: STATE_LOADING_SERVER, Description: "User clicks a server in the UI"},
				{To: STATE_CHOSEN_SERVER, Description: "The server has been chosen"},
				{To: STATE_SEARCH_SERVER, Description: "The user is trying to choose a new server in the UI"},
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
					To:          STATE_ASK_LOCATION,
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
				{To: STATE_NO_SERVER, Description: "Client wants to go back to the main screen"},
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
	returnedFSM.Init(STATE_DEREGISTERED, states, callback, directory, GetStateName, debug)
	return returnedFSM
}

type FSMDeregisteredError struct{}

func (e FSMDeregisteredError) CustomError() *types.WrappedErrorMessage {
	return types.NewWrappedError(
		"Client not registered with the GO library",
		errors.New(
			"the current FSM state is deregistered, but the function needs a state that is not deregistered",
		),
	)
}

type FSMWrongStateTransitionError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e FSMWrongStateTransitionError) CustomError() *types.WrappedErrorMessage {
	return types.NewWrappedError(
		"Wrong FSM transition",
		fmt.Errorf(
			"wrong FSM state, got: %s, want: a state with a transition to: %s",
			GetStateName(e.Got),
			GetStateName(e.Want),
		),
	)
}

type FSMWrongStateError struct {
	Got  FSMStateID
	Want FSMStateID
}

func (e FSMWrongStateError) CustomError() *types.WrappedErrorMessage {
	return types.NewWrappedError(
		"Wrong FSM State",
		fmt.Errorf(
			"wrong FSM state, got: %s, want: %s",
			GetStateName(e.Got),
			GetStateName(e.Want),
		),
	)
}


// SetSearchServer sets the FSM to the SEARCH_SERVER state.
// This indicates that the user wants to search for a new server.
// Returns an error if this state transition is not possible.
func (client *Client) SetSearchServer() error {
	if !client.FSM.HasTransition(STATE_SEARCH_SERVER) {
		return client.handleError(
			"failed to set search server",
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: STATE_SEARCH_SERVER,
			}.CustomError(),
		)
	}

	client.FSM.GoTransition(STATE_SEARCH_SERVER)
	return nil
}

// SetConnected sets the FSM to the CONNECTED state.
// This indicates that the VPN is connected to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetConnected() error {
	errorMessage := "failed to set connected"
	if client.InFSMState(STATE_CONNECTED) {
		// already connected, show no error
		client.Logger.Warning("Already connected")
		return nil
	}
	if !client.FSM.HasTransition(STATE_CONNECTED) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: STATE_CONNECTED,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(STATE_CONNECTED, currentServer)
	return nil
}

// SetConnecting sets the FSM to the CONNECTING state.
// This indicates that the VPN is currently connecting to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetConnecting() error {
	errorMessage := "failed to set connecting"
	if client.InFSMState(STATE_CONNECTING) {
		// already loading connection, show no error
		client.Logger.Warning("Already connecting")
		return nil
	}
	if !client.FSM.HasTransition(STATE_CONNECTING) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: STATE_CONNECTING,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(STATE_CONNECTING, currentServer)
	return nil
}

// SetDisconnecting sets the FSM to the DISCONNECTING state.
// This indicates that the VPN is currently disconnecting from the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetDisconnecting() error {
	errorMessage := "failed to set disconnecting"
	if client.InFSMState(STATE_DISCONNECTING) {
		// already disconnecting, show no error
		client.Logger.Warning("Already disconnecting")
		return nil
	}
	if !client.FSM.HasTransition(STATE_DISCONNECTING) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: STATE_DISCONNECTING,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(STATE_DISCONNECTING, currentServer)
	return nil
}

// SetDisconnected sets the FSM to the DISCONNECTED state.
// This indicates that the VPN is currently disconnected from the server.
// This also sends the /disconnect API call to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetDisconnected(cleanup bool) error {
	errorMessage := "failed to set disconnected"
	if client.InFSMState(STATE_DISCONNECTED) {
		// already disconnected, show no error
		client.Logger.Warning("Already disconnected")
		return nil
	}
	if !client.FSM.HasTransition(STATE_DISCONNECTED) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: STATE_DISCONNECTED,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	if cleanup {
		// Do the /disconnect API call and go to disconnected after...
		server.Disconnect(currentServer)
	}

	client.FSM.GoTransitionWithData(STATE_DISCONNECTED, currentServer)

	return nil
}

// goBackInternal uses the public go back but logs an error if it happened.
func (client *Client) goBackInternal() {
	goBackErr := client.GoBack()
	if goBackErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed going back, error: %s",
				types.GetErrorTraceback(goBackErr),
			),
		)
	}
}

// GoBack transitions the FSM back to the previous UI state, for now this is always the NO_SERVER state.
func (client *Client) GoBack() error {
	errorMessage := "failed to go back"
	if client.InFSMState(STATE_DEREGISTERED) {
		return client.handleError(
			errorMessage,
			FSMDeregisteredError{}.CustomError(),
		)
	}

	// FIXME: Abitrary back transitions don't work because we need the approriate data
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers)
	return nil
}

// CancelOAuth cancels OAuth if one is in progress.
// If OAuth is not in progress, it returns an error.
// An error is also returned if OAuth is in progress but it fails to cancel it.
func (client *Client) CancelOAuth() error {
	errorMessage := "failed to cancel OAuth"
	if !client.InFSMState(STATE_OAUTH_STARTED) {
		return client.handleError(
			errorMessage,
			FSMWrongStateError{
				Got:  client.FSM.Current,
				Want: STATE_OAUTH_STARTED,
			}.CustomError(),
		)
	}

	currentServer, serverErr := client.Servers.GetCurrentServer()
	if serverErr != nil {
		return client.handleError(errorMessage, serverErr)
	}
	server.CancelOAuth(currentServer)
	return nil
}


// InFSMState is a helper to check if the FSM is in state `checkState`.
func (client *Client) InFSMState(checkState FSMStateID) bool {
	return client.FSM.InState(checkState)
}
