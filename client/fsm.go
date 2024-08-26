package client

import (
	"fmt"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/log"
)

type (
	// FSMStateID is an alias to the fsm state ID type
	FSMStateID = fsm.StateID
	// FSMStates is an alias to the fsm states type
	FSMStates = fsm.States
	// FSMState is an alias to the fsm state type
	FSMState = fsm.State
	// FSMTransition is an alias to the fsm transition type
	FSMTransition = fsm.Transition
)

const (
	// StateDeregistered is the state where we are deregistered
	StateDeregistered FSMStateID = iota

	// StateMain is the main state
	StateMain

	// StateAddingServer is the state where a server is being added
	StateAddingServer

	// StateOAuthStarted means the state where the OAuth procedure is triggered
	StateOAuthStarted

	// StateGettingConfig is the state a VPN config is being obtained
	StateGettingConfig

	// StateAskLocation is the state where a secure internet location is being asked
	StateAskLocation

	// StateAskProfile is the state where a profile is being asked for
	StateAskProfile

	// StateGotConfig is the state where a config is obtained
	StateGotConfig

	// StateConnecting is the state where the VPN is connecting
	StateConnecting

	// StateConnected is the state where the VPN is connected
	StateConnected

	// StateDisconnecting is the state where the VPN is disconnecting
	StateDisconnecting

	// StateDisconnected is the state where the VPN is disconnected
	StateDisconnected
)

// GetStateName gets the State name for state `s`
func GetStateName(s FSMStateID) string {
	switch s {
	case StateDeregistered:
		return "Deregistered"
	case StateMain:
		return "Main"
	case StateAddingServer:
		return "AddingServer"
	case StateOAuthStarted:
		return "OAuthStarted"
	case StateGettingConfig:
		return "GettingConfig"
	case StateAskLocation:
		return "AskLocation"
	case StateAskProfile:
		return "AskProfile"
	case StateGotConfig:
		return "GotConfig"
	case StateConnecting:
		return "Connecting"
	case StateConnected:
		return "Connected"
	case StateDisconnecting:
		return "Disconnecting"
	case StateDisconnected:
		return "Disconnected"
	default:
		panic(fmt.Sprintf("unknown conversion of state: %d to string", s))
	}
}

func newFSM(
	callback func(FSMStateID, FSMStateID, interface{}) bool,
) fsm.FSM {
	states := FSMStates{
		StateDeregistered: FSMState{
			Transitions: []FSMTransition{
				{To: StateMain, Description: "Register"},
			},
		},
		StateMain: FSMState{
			Transitions: []FSMTransition{
				{To: StateDeregistered, Description: "Deregister"},
				{To: StateAddingServer, Description: "Add a server"},
				{To: StateGettingConfig, Description: "Get a VPN config"},
				{To: StateConnected, Description: "Already connected"},
			},
		},
		StateAddingServer: FSMState{
			Transitions: []FSMTransition{
				{To: StateOAuthStarted, Description: "Authorize"},
			},
		},
		StateOAuthStarted: FSMState{
			Transitions: []FSMTransition{
				{To: StateMain, Description: "Authorized"},
				{To: StateDisconnected, Description: "Cancel, was disconnected"},
				{To: StateGotConfig, Description: "Cancel, was got config"},
			},
		},
		StateGettingConfig: FSMState{
			Transitions: []FSMTransition{
				{To: StateAskLocation, Description: "Invalid location"},
				{To: StateAskProfile, Description: "Invalid or no profile"},
				{To: StateDisconnected, Description: "Go back to disconnected"},
				{To: StateGotConfig, Description: "Successfully got a configuration"},
				{To: StateOAuthStarted, Description: "Authorize"},
			},
		},
		StateAskLocation: FSMState{
			Transitions: []FSMTransition{
				{To: StateGettingConfig, Description: "Location chosen"},
			},
		},
		StateAskProfile: FSMState{
			Transitions: []FSMTransition{
				{To: StateGettingConfig, Description: "Profile chosen"},
			},
		},
		StateGotConfig: FSMState{
			Transitions: []FSMTransition{
				{To: StateGettingConfig, Description: "Get a VPN config again"},
				{To: StateConnecting, Description: "VPN is connecting"},
				{To: StateOAuthStarted, Description: "Renew"},
			},
		},
		StateConnecting: FSMState{
			Transitions: []FSMTransition{
				{To: StateConnected, Description: "VPN is connected"},
				{To: StateDisconnecting, Description: "Cancel connecting"},
			},
		},
		StateConnected: FSMState{
			Transitions: []FSMTransition{
				{To: StateDisconnecting, Description: "VPN is disconnecting"},
			},
		},
		StateDisconnecting: FSMState{
			Transitions: []FSMTransition{
				{To: StateDisconnected, Description: "VPN is disconnected"},
				{To: StateConnected, Description: "Cancel disconnecting"},
			},
		},
		StateDisconnected: FSMState{
			Transitions: []FSMTransition{
				{To: StateConnecting, Description: "Connect with existing config"},
				{To: StateGettingConfig, Description: "Connect with a new config"},
				{To: StateOAuthStarted, Description: "Renew"},
			},
		},
	}

	return fsm.NewFSM(StateMain, states, callback, GetStateName)
}

// SetState sets the state for the client FSM to `state`
func (c *Client) SetState(state FSMStateID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	curr := c.FSM.Current
	_, err := c.FSM.GoTransition(state)
	if err != nil {
		// self-transitions are only debug errors
		if c.FSM.InState(state) {
			log.Logger.Debugf("attempt an invalid self-transition: %s", c.FSM.GetStateName(state))
			return nil
		}
		return i18nerr.WrapInternalf(err, "Failed internal state transition requested by the client from: '%s' to '%s'", GetStateName(curr), GetStateName(state))
	}
	return nil
}

// InState returns whether or not the client is in state `state`
func (c *Client) InState(state FSMStateID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.FSM.InState(state)
}
