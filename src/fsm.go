package eduvpn

import (
	"fmt"
	"os"
)

type FSMStateID int8

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
	case AUTHENTICATED:
		return "Authenticated"
	case CONNECTED:
		return "Connected"
	default:
		panic("unknown conversion of state to string")
	}
}

type (
	FSMTransitions []FSMStateID
	FSMStates      map[FSMStateID]FSMTransitions
)

type FSM struct {
	States  FSMStates
	Current FSMStateID
}

func (eduvpn *VPNState) HasTransition(check FSMStateID) bool {
	for _, transition_state := range eduvpn.FSM.States[eduvpn.FSM.Current] {
		if transition_state == check {
			return true
		}
	}

	return false
}

func (eduvpn *VPNState) InState(check FSMStateID) bool {
	return check == eduvpn.FSM.Current
}

func (eduvpn *VPNState) writeGraph() {
	graph := eduvpn.GenerateGraph()

	f, err := os.Create("debug.graph")
	if err != nil {
		eduvpn.Log(LOG_INFO, fmt.Sprintf("Failed to write debug fsm graph with error %v", err))
	}

	defer f.Close()

	f.WriteString(graph)
}

func (eduvpn *VPNState) GoTransition(newState FSMStateID, data string) bool {
	ok := eduvpn.HasTransition(newState)

	if ok {
		oldState := eduvpn.FSM.Current
		eduvpn.FSM.Current = newState
		if eduvpn.Debug {
			eduvpn.writeGraph()
		}
		eduvpn.StateCallback(oldState.String(), newState.String(), data)
	}

	return ok
}

func (eduvpn *VPNState) GenerateGraph() string {
	graph := `digraph eduvpn_fsm {
nodesep = 2;
rankdir = LR;`
	graph += "\nnode[color=blue]; " + eduvpn.FSM.Current.String() + ";\n"
	graph += "node [color=black];\n"
	for state, transitions := range eduvpn.FSM.States {
		for _, transition_state := range transitions {
			graph += state.String() + " -> " + transition_state.String() + "\n"
		}
	}
	graph += "}"
	return graph
}

func (eduvpn *VPNState) InitializeFSM() {
	eduvpn.FSM = &FSM{
		States: FSMStates {
			DEREGISTERED: {NO_SERVER},
			NO_SERVER: {CHOSEN_SERVER},
			CHOSEN_SERVER: {AUTHENTICATED, OAUTH_STARTED},
			OAUTH_STARTED: {AUTHENTICATED},
			AUTHENTICATED: {CONNECTED},
			CONNECTED: {AUTHENTICATED},
		},
		Current: DEREGISTERED,
	}
}
