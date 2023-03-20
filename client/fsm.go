package client

import (
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/log"
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

	// StateChosenLocation means the user has selected a Secure Internet location
	StateChosenLocation

	// StateLoadingServer means we are loading the server details.
	StateLoadingServer

	// StateChosenServer means the user has chosen a server to connect to and the server is initialized.
	StateChosenServer

	// StateOAuthStarted means the OAuth process has started.
	StateOAuthStarted

	// StateAuthorized means the OAuth process has finished and the user is now authorized with the server.
	StateAuthorized

	// StateRequestConfig means the user has requested a config for connecting.
	StateRequestConfig

	// StateAskProfile means the go code is asking for a profile selection from the UI.
	StateAskProfile

	// StateChosenProfile means a profile has been chosen
	StateChosenProfile

	// StateGotConfig means a VPN configuration has been obtained
	StateGotConfig
)

func GetStateName(s FSMStateID) string {
	switch s {
	case StateDeregistered:
		return "Deregistered"
	case StateNoServer:
		return "No_Server"
	case StateAskLocation:
		return "Ask_Location"
	case StateLoadingServer:
		return "Loading_Server"
	case StateChosenServer:
		return "Chosen_Server"
	case StateChosenLocation:
		return "Chosen_Location"
	case StateOAuthStarted:
		return "OAuth_Started"
	case StateRequestConfig:
		return "Request_Config"
	case StateAskProfile:
		return "Ask_Profile"
	case StateChosenProfile:
		return "Chosen_Profile"
	case StateAuthorized:
		return "Authorized"
	case StateGotConfig:
		return "Got_Config"
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
			},
		},
		StateAskLocation: FSMState{
			Transitions: []FSMTransition{
				{To: StateChosenLocation, Description: "Location chosen"},
				{To: StateNoServer, Description: "Go back or Error"},
			},
		},
		StateChosenLocation: FSMState{
			Transitions: []FSMTransition{
				{To: StateChosenServer, Description: "Server has been chosen"},
				{To: StateNoServer, Description: "Go back or Error"},
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
				{To: StateChosenProfile, Description: "Only one profile or profile already chosen"},
				{To: StateNoServer, Description: "Cancel or Error"},
				{To: StateOAuthStarted, Description: "Re-authorize"},
			},
		},
		StateAskProfile: FSMState{
			Transitions: []FSMTransition{
				{To: StateNoServer, Description: "Cancel or Error"},
				{To: StateChosenProfile, Description: "Profile has been chosen"},
			},
		},
		StateChosenProfile: FSMState{
			Transitions: []FSMTransition{
				{To: StateNoServer, Description: "Cancel or Error"},
				{To: StateGotConfig, Description: "Config has been obtained"},
			},
		},
		StateGotConfig: FSMState{
			Transitions: []FSMTransition{
				{To: StateNoServer, Description: "Choose a new server"},
				{To: StateLoadingServer, Description: "Get a new configuration"},
			},
		},
	}
	returnedFSM := fsm.FSM{}
	returnedFSM.Init(StateDeregistered, states, callback, directory, GetStateName, debug)
	return returnedFSM
}

// GoBack transitions the FSM back to the previous UI state, for now this is always the NO_SERVER state.
func (c *Client) GoBack() error {
	if c.InFSMState(StateDeregistered) {
		err := errors.Errorf("fsm attempt going back from 'StateDeregistered'")
		c.logError(err)
		return err
	}

	// FIXME: Arbitrary back transitions don't work because we need the appropriate data
	c.FSM.GoTransition(StateNoServer)
	return nil
}

// goBackInternal uses the public go back but logs an error if it happened.
func (c *Client) goBackInternal() {
	err := c.GoBack()
	if err != nil {
		// TODO(jwijenbergh): Bit suspicious - logging level INFO, yet stacktrace logged.
		log.Logger.Infof("failed going back: %s\nstacktrace:\n%s", err.Error(), err.(*errors.Error).ErrorStack())
	}
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
