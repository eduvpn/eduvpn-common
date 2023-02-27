package client

import (
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/go-errors/errors"
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
				{
					To:          StateSearchServer,
					Description: "The user is trying to choose a new server in the UI",
				},
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
			Transitions: []FSMTransition{
				{To: StateDisconnecting, Description: "App wants to disconnect"},
			},
		},
	}
	returnedFSM := fsm.FSM{}
	returnedFSM.Init(StateDeregistered, states, callback, directory, GetStateName, debug)
	return returnedFSM
}

// SetSearchServer sets the FSM to the SEARCH_SERVER state.
// This indicates that the user wants to search for a new server.
// Returns an error if this state transition is not possible.
func (c *Client) SetSearchServer() error {
	if err := c.FSM.CheckTransition(StateSearchServer); err != nil {
		c.logError(err)
		return err
	}

	// TODO(jwijenbergh): Should we handle `false` returned value here?
	c.FSM.GoTransition(StateSearchServer)
	return nil
}

// SetConnected sets the FSM to the CONNECTED state.
// This indicates that the VPN is connected to the server.
// Returns an error if this state transition is not possible.
func (c *Client) SetConnected() error {
	if c.InFSMState(StateConnected) {
		// already connected, show no error
		return nil
	}

	if err := c.FSM.CheckTransition(StateConnected); err != nil {
		c.logError(err)
		return err
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
		return err
	}

	c.FSM.GoTransitionWithData(StateConnected, srv)
	return nil
}

// SetConnecting sets the FSM to the CONNECTING state.
// This indicates that the VPN is currently connecting to the server.
// Returns an error if this state transition is not possible.
func (c *Client) SetConnecting() error {
	if c.InFSMState(StateConnecting) {
		// already loading connection, show no error
		c.Logger.Warningf("Already connecting")
		return nil
	}
	if err := c.FSM.CheckTransition(StateConnecting); err != nil {
		c.logError(err)
		return err
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
		return err
	}

	c.FSM.GoTransitionWithData(StateConnecting, srv)
	return nil
}

// SetDisconnecting sets the FSM to the DISCONNECTING state.
// This indicates that the VPN is currently disconnecting from the server.
// Returns an error if this state transition is not possible.
func (c *Client) SetDisconnecting() error {
	if c.InFSMState(StateDisconnecting) {
		// already disconnecting, show no error
		c.Logger.Warningf("Already disconnecting")
		return nil
	}
	if err := c.FSM.CheckTransition(StateDisconnecting); err != nil {
		c.logError(err)
		return err
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
		return err
	}

	c.FSM.GoTransitionWithData(StateDisconnecting, srv)
	return nil
}

// SetDisconnected sets the FSM to the DISCONNECTED state.
// This indicates that the VPN is currently disconnected from the server.
// This also sends the /disconnect API call to the server.
// Returns an error if this state transition is not possible.
func (c *Client) SetDisconnected() error {
	if c.InFSMState(StateDisconnected) {
		// already disconnected, show no error
		c.Logger.Warningf("Already disconnected")
		return nil
	}
	if err := c.FSM.CheckTransition(StateDisconnected); err != nil {
		c.logError(err)
		return err
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
		return err
	}

	c.FSM.GoTransitionWithData(StateDisconnected, srv)

	return nil
}

// goBackInternal uses the public go back but logs an error if it happened.
func (c *Client) goBackInternal() {
	err := c.GoBack()
	if err != nil {
		// TODO(jwijenbergh): Bit suspicious - logging level INFO, yet stacktrace logged.
		c.Logger.Infof("failed going back: %s\nstacktrace:\n%s", err.Error(), err.(*errors.Error).ErrorStack())
	}
}

// GoBack transitions the FSM back to the previous UI state, for now this is always the NO_SERVER state.
func (c *Client) GoBack() error {
	if c.InFSMState(StateDeregistered) {
		err := errors.Errorf("fsm attempt going back from 'StateDeregistered'")
		c.logError(err)
		return err
	}

	// FIXME: Arbitrary back transitions don't work because we need the appropriate data
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)
	return nil
}

// CancelOAuth cancels OAuth if one is in progress.
// If OAuth is not in progress, it returns an error.
// An error is also returned if OAuth is in progress, but it fails to cancel it.
func (c *Client) CancelOAuth() error {
	if !c.InFSMState(StateOAuthStarted) {
		return errors.Errorf("fsm attempt cancelling OAuth while in '%v'", c.FSM.Current)
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
	} else {
		server.CancelOAuth(srv)
	}
	return err
}

// InFSMState is a helper to check if the FSM is in state `checkState`.
func (c *Client) InFSMState(checkState FSMStateID) bool {
	return c.FSM.InState(checkState)
}
