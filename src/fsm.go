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
	To FSMStateID
	Description string
}

type (
	FSMTransitions []FSMTransition
	FSMStates      map[FSMStateID]FSMTransitions
)

type FSM struct {
	States  FSMStates
	Current FSMStateID
}

func (eduvpn *VPNState) HasTransition(check FSMStateID) bool {
	for _, transition_state := range eduvpn.FSM.States[eduvpn.FSM.Current] {
		if transition_state.To == check {
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

func (eduvpn *VPNState) GoTransition(newState FSMStateID, data string) (bool, string) {
	ok := eduvpn.HasTransition(newState)

	received := ""

	if ok {
		oldState := eduvpn.FSM.Current
		eduvpn.FSM.Current = newState
		if eduvpn.Debug {
			eduvpn.writeGraph()
		}
		received = eduvpn.StateCallback(oldState.String(), newState.String(), data)
	}

	return ok, received
}

func (eduvpn *VPNState) generateDotGraph() string {
	graph := `digraph eduvpn_fsm {
nodesep = 2;
remincross = false;
`
	graph += "node[color=blue]; " + eduvpn.FSM.Current.String() + ";\n"
	graph += "node [color=black];\n"
	for state, transitions := range eduvpn.FSM.States {
		for _, transition := range transitions {
			graph += state.String() + " -> " + transition.To.String() + " [label=\"" + transition.Description + "\"]\n"
		}
	}
	graph += "}"
	return graph
}

func (eduvpn *VPNState) generateMermaidGraph() string {
	graph := "graph TD\n"
	graph += "style " + eduvpn.FSM.Current.String() + " fill:cyan\n"
	for state, transitions := range eduvpn.FSM.States {
		for _, transition := range transitions {
			graph += state.String() + "(" + state.String() + ") " + "-->|" + transition.Description + "| " + transition.To.String() + "\n"
		}
	}
	return graph
}

func (eduvpn *VPNState) GenerateGraph() string {
	return eduvpn.generateMermaidGraph()
}

func (eduvpn *VPNState) InitializeFSM() {
	eduvpn.FSM = &FSM{
		States: FSMStates {
			DEREGISTERED: {{NO_SERVER, "Client registers"}},
			NO_SERVER: {{CHOSEN_SERVER, "User chooses a server"}},
			CHOSEN_SERVER: {{AUTHENTICATED, "Found tokens in config"}, {OAUTH_STARTED, "No tokens found in config"}},
			OAUTH_STARTED: {{AUTHENTICATED, "User authorizes with browser"}},
			AUTHENTICATED: {{CONNECTED, "OS reports connected"}, {OAUTH_STARTED, "Re-authenticate with OAuth"}, {ASK_PROFILE, "Connect, multiple profiles detected"}},
			ASK_PROFILE: {{CONNECTED, "OS reports connected"}},
			CONNECTED: {{AUTHENTICATED, "OS reports disconnected"}},
		},
		Current: DEREGISTERED,
	}
}
