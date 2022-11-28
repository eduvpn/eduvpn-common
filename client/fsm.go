package client

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/types"
)

type (
	FSMStateID    = fsm.StateID
	FSMStates     = fsm.States
	FSMState      = fsm.State
	FSMTransition = fsm.Transition
)

const (
	// StateDeregistered means the app is not registered with the wrapper.
	StateDeregistered FSMStateID = iota

	// StateNoServer means the user has not chosen a server yet.
	StateNoServer

	// StateAskLocation means the user selected a Secure Internet server but needs to choose a location.
	StateAskLocation

	// StateSearchServer means the user is currently selecting a server in the UI.
	StateSearchServer

	// StateLoadingServer means we are loading the server details.
	StateLoadingServer

	// StateChosenServer means the user has chosen a server to connect to.
	StateChosenServer

	// StateOAuthStarted means the OAuth process has started.
	StateOAuthStarted

	// StateAuthorized means the OAuth process has finished and the user is now authorized with the server.
	StateAuthorized

	// StateRequestConfig means the user has requested a config for connecting.
	StateRequestConfig

	// StateAskProfile means the go code is asking for a profile selection from the UI.
	StateAskProfile

	// StateDisconnected means the user has gotten a config for a server but is not connected yet.
	StateDisconnected

	// StateDisconnecting means the OS is disconnecting and the Go code is doing the /disconnect.
	StateDisconnecting

	// StateConnecting means the OS is establishing a connection to the server.
	StateConnecting

	// StateConnected means the user has been connected to the server.
	StateConnected
)

func GetStateName(s FSMStateID) string {
	switch s {
	case StateDeregistered:
		return "Deregistered"
	case StateNoServer:
		return "No_Server"
	case StateAskLocation:
		return "Ask_Location"
	case StateSearchServer:
		return "Search_Server"
	case StateLoadingServer:
		return "Loading_Server"
	case StateChosenServer:
		return "Chosen_Server"
	case StateOAuthStarted:
		return "OAuth_Started"
	case StateDisconnected:
		return "Disconnected"
	case StateRequestConfig:
		return "Request_Config"
	case StateAskProfile:
		return "Ask_Profile"
	case StateAuthorized:
		return "Authorized"
	case StateDisconnecting:
		return "Disconnecting"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
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
		StateDeregistered: FSMState{
			Transitions: []FSMTransition{{To: StateNoServer, Description: "Client registers"}},
		},
		StateNoServer: FSMState{
			Transitions: []FSMTransition{
				{To: StateNoServer, Description: "Reload list"},
				{To: StateLoadingServer, Description: "User clicks a server in the UI"},
				{To: StateChosenServer, Description: "The server has been chosen"},
				{To: StateSearchServer, Description: "The user is trying to choose a new server in the UI"},
				{To: StateConnected, Description: "The user is already connected"},
				{To: StateAskLocation, Description: "Change the location in the main screen"},
			},
		},
		StateSearchServer: FSMState{
			Transitions: []FSMTransition{
				{To: StateLoadingServer, Description: "User clicks a server in the UI"},
				{To: StateNoServer, Description: "Cancel or Error"},
			},
		},
		StateAskLocation: FSMState{
			Transitions: []FSMTransition{
				{To: StateChosenServer, Description: "Location chosen"},
				{To: StateNoServer, Description: "Go back or Error"},
				{To: StateSearchServer, Description: "Cancel or Error"},
			},
		},
		StateLoadingServer: FSMState{
			Transitions: []FSMTransition{
				{To: StateChosenServer, Description: "Server info loaded"},
				{
					To:          StateAskLocation,
					Description: "User chooses a Secure Internet server but no location is configured",
				},
				{To: StateNoServer, Description: "Go back or Error"},
			},
		},
		StateChosenServer: FSMState{
			Transitions: []FSMTransition{
				{To: StateAuthorized, Description: "Found tokens in config"},
				{To: StateOAuthStarted, Description: "No tokens found in config"},
			},
		},
		StateOAuthStarted: FSMState{
			Transitions: []FSMTransition{
				{To: StateAuthorized, Description: "User authorizes with browser"},
				{To: StateNoServer, Description: "Go back or Error"},
				{To: StateSearchServer, Description: "Cancel or Error"},
			},
		},
		StateAuthorized: FSMState{
			Transitions: []FSMTransition{
				{To: StateOAuthStarted, Description: "Re-authorize with OAuth"},
				{To: StateRequestConfig, Description: "Client requests a config"},
				{To: StateNoServer, Description: "Client wants to go back to the main screen"},
			},
		},
		StateRequestConfig: FSMState{
			Transitions: []FSMTransition{
				{To: StateAskProfile, Description: "Multiple profiles found and no profile chosen"},
				{To: StateDisconnected, Description: "Only one profile or profile already chosen"},
				{To: StateNoServer, Description: "Cancel or Error"},
				{To: StateOAuthStarted, Description: "Re-authorize"},
			},
		},
		StateAskProfile: FSMState{
			Transitions: []FSMTransition{
				{To: StateDisconnected, Description: "User chooses profile"},
				{To: StateNoServer, Description: "Cancel or Error"},
				{To: StateSearchServer, Description: "Cancel or Error"},
			},
		},
		StateDisconnected: FSMState{
			Transitions: []FSMTransition{
				{To: StateConnecting, Description: "OS reports it is trying to connect"},
				{To: StateRequestConfig, Description: "User reconnects"},
				{To: StateNoServer, Description: "User wants to choose a new server"},
				{To: StateOAuthStarted, Description: "Re-authorize with OAuth"},
			},
		},
		StateDisconnecting: FSMState{
			Transitions: []FSMTransition{
				{To: StateDisconnected, Description: "Cancel or Error"},
				{To: StateDisconnected, Description: "Done disconnecting"},
			},
		},
		StateConnecting: FSMState{
			Transitions: []FSMTransition{
				{To: StateDisconnected, Description: "Cancel or Error"},
				{To: StateConnected, Description: "Done connecting"},
			},
		},
		StateConnected: FSMState{
			Transitions: []FSMTransition{{To: StateDisconnecting, Description: "App wants to disconnect"}},
		},
	}
	returnedFSM := fsm.FSM{}
	returnedFSM.Init(StateDeregistered, states, callback, directory, GetStateName, debug)
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
	if !client.FSM.HasTransition(StateSearchServer) {
		return client.handleError(
			"failed to set search server",
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: StateSearchServer,
			}.CustomError(),
		)
	}

	client.FSM.GoTransition(StateSearchServer)
	return nil
}

// SetConnected sets the FSM to the CONNECTED state.
// This indicates that the VPN is connected to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetConnected() error {
	errorMessage := "failed to set connected"
	if client.InFSMState(StateConnected) {
		// already connected, show no error
		client.Logger.Warning("Already connected")
		return nil
	}
	if !client.FSM.HasTransition(StateConnected) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: StateConnected,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(StateConnected, currentServer)
	return nil
}

// SetConnecting sets the FSM to the CONNECTING state.
// This indicates that the VPN is currently connecting to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetConnecting() error {
	errorMessage := "failed to set connecting"
	if client.InFSMState(StateConnecting) {
		// already loading connection, show no error
		client.Logger.Warning("Already connecting")
		return nil
	}
	if !client.FSM.HasTransition(StateConnecting) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: StateConnecting,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(StateConnecting, currentServer)
	return nil
}

// SetDisconnecting sets the FSM to the DISCONNECTING state.
// This indicates that the VPN is currently disconnecting from the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetDisconnecting() error {
	errorMessage := "failed to set disconnecting"
	if client.InFSMState(StateDisconnecting) {
		// already disconnecting, show no error
		client.Logger.Warning("Already disconnecting")
		return nil
	}
	if !client.FSM.HasTransition(StateDisconnecting) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: StateDisconnecting,
			}.CustomError(),
		)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	client.FSM.GoTransitionWithData(StateDisconnecting, currentServer)
	return nil
}

// SetDisconnected sets the FSM to the DISCONNECTED state.
// This indicates that the VPN is currently disconnected from the server.
// This also sends the /disconnect API call to the server.
// Returns an error if this state transition is not possible.
func (client *Client) SetDisconnected(cleanup bool) error {
	errorMessage := "failed to set disconnected"
	if client.InFSMState(StateDisconnected) {
		// already disconnected, show no error
		client.Logger.Warning("Already disconnected")
		return nil
	}
	if !client.FSM.HasTransition(StateDisconnected) {
		return client.handleError(
			errorMessage,
			FSMWrongStateTransitionError{
				Got:  client.FSM.Current,
				Want: StateDisconnected,
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

	client.FSM.GoTransitionWithData(StateDisconnected, currentServer)

	return nil
}

// goBackInternal uses the public go back but logs an error if it happened.
func (client *Client) goBackInternal() {
	goBackErr := client.GoBack()
	if goBackErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed going back, error: %s",
				types.ErrorTraceback(goBackErr),
			),
		)
	}
}

// GoBack transitions the FSM back to the previous UI state, for now this is always the NO_SERVER state.
func (client *Client) GoBack() error {
	errorMessage := "failed to go back"
	if client.InFSMState(StateDeregistered) {
		return client.handleError(
			errorMessage,
			FSMDeregisteredError{}.CustomError(),
		)
	}

	// FIXME: Abitrary back transitions don't work because we need the approriate data
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	return nil
}

// CancelOAuth cancels OAuth if one is in progress.
// If OAuth is not in progress, it returns an error.
// An error is also returned if OAuth is in progress but it fails to cancel it.
func (client *Client) CancelOAuth() error {
	errorMessage := "failed to cancel OAuth"
	if !client.InFSMState(StateOAuthStarted) {
		return client.handleError(
			errorMessage,
			FSMWrongStateError{
				Got:  client.FSM.Current,
				Want: StateOAuthStarted,
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
