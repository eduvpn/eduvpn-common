package eduvpn

import (
	"errors"
	"os"
)

type FSMStateID int8

const (
	// Registered means the app is registered with the wrapper
	APP_REGISTERED FSMStateID = iota

	// Deregistered means the app is not registered with the wrapper
	APP_DEREGISTERED

	// We have the states where a server is chosen or not
	// When no server is chosen, we have no substate
	CONFIG_NOSERVER

	// When a server is chosen we have the remaining states as substates
	CONFIG_CHOSENSERVER

	// The states for when the server is authenticated
	// The SERVER_AUTHENTICATED is the parent state
	// While SERVER_CONNECTED and SERVER_DISCONNECTED are substatse
	SERVER_AUTHENTICATED
	SERVER_CONNECTED
	SERVER_DISCONNECTED

	// The states for when the server is not authenticated
	// The SERVER_NOT_AUTHENTICATED is the parent state
	// While SERVER_INITIALIZED, SERVER_OAUTH_STARTED and SERVER_OAUTH_FINISHED are substates
	SERVER_NOT_AUTHENTICATED
	SERVER_INITIALIZED
	SERVER_OAUTH_STARTED
	SERVER_OAUTH_FINISHED
)

func (s FSMStateID) String() string {
	switch s {
	case APP_REGISTERED:
		return "APP_REGISTERED"
	case APP_DEREGISTERED:
		return "APP_DEREGISTERED"
	case CONFIG_NOSERVER:
		return "CONFIG_NOSERVER"
	case CONFIG_CHOSENSERVER:
		return "CONFIG_CHOSENSERVER"
	case SERVER_AUTHENTICATED:
		return "SERVER_AUTHENTICATED"
	case SERVER_CONNECTED:
		return "SERVER_CONNECTED"
	case SERVER_DISCONNECTED:
		return "SERVER_DISCONNECTED"
	case SERVER_NOT_AUTHENTICATED:
		return "SERVER_NOT_AUTHENTICATED"
	case SERVER_INITIALIZED:
		return "SERVER_INITIALIZED"
	case SERVER_OAUTH_STARTED:
		return "SERVER_OAUTH_STARTED"
	case SERVER_OAUTH_FINISHED:
		return "SERVER_OAUTH_FINISHED"
	default:
		panic("unknown conversion of state to string")
	}
}

type (
	FSMStates      map[FSMStateID]*FSMState
	FSMTransitions []FSMStateID
)

type FSMState struct {
	Sub        *FSM
	Transition FSMTransitions

	// When Locked=True it cannot go to the parent state and transition away
	Locked bool
}

type FSM struct {
	States  FSMStates
	Current FSMStateID
}

func (fsmState *FSMState) hasTransition(check FSMStateID) bool {
	for _, state := range fsmState.Transition {
		if state == check {
			return true
		}
	}
	return false
}

func (eduvpn *VPNState) getCurrentState() (*FSMState, error) {
	state, hasState := eduvpn.FSM.States[eduvpn.FSM.Current]

	if !hasState {
		return nil, errors.New("Cannot get current state")
	}

	return state, nil
}

func FindFSMState(state FSMStateID, fsm *FSM) *FSM {
	if fsm == nil {
		return nil
	}

	// Check if the state is in the current fsm
	retrievedState, hasState := fsm.States[state]

	// Otherwise we need to go to the sub states
	if !hasState || retrievedState == nil {
		return FindFSMState(state, fsm.States[fsm.Current].Sub)
	} else {
		return fsm
	}
}

func (eduvpn *VPNState) IsInFSMState(check FSMStateID) bool {
	return eduvpn.FSM.Current == check
}

func (eduvpn *VPNState) findTransition(check FSMStateID) (*FSM, bool) {
	fsm := FindFSMState(check, eduvpn.FSM)

	if fsm == nil {
		return nil, false
	}

	subStates := fsm.States[fsm.Current].Sub

	if subStates != nil {
		if subStates.States[subStates.Current].Locked {
			return nil, false
		}
	}

	for _, val := range fsm.States[fsm.Current].Transition {
		if val == check {
			return fsm, true
		}
	}

	return nil, false
}

func (eduvpn *VPNState) HasTransition(check FSMStateID) bool {
	fsm, ok := eduvpn.findTransition(check)

	return ok && fsm != nil
}

func (eduvpn *VPNState) InState(check FSMStateID) bool {
	fsm := FindFSMState(check, eduvpn.FSM)

	if fsm == nil {
		return false
	}

	return fsm.Current == check
}

func (eduvpn *VPNState) writeGraph() {
	graph := eduvpn.GenerateGraph()

	f, err := os.Create("debug.graph")

	if err != nil {
		panic(err)
	}

	defer f.Close()

	f.WriteString(graph)
}

func (eduvpn *VPNState) GoTransition(newState FSMStateID, data string) bool {
	fsm, ok := eduvpn.findTransition(newState)

	if ok {
		oldState := fsm.Current
		fsm.Current = newState
		if eduvpn.Debug {
			eduvpn.writeGraph()
		}
		eduvpn.StateCallback(oldState.String(), newState.String(), data)
	}

	return ok
}

func getGraphviz(fsm *FSM, graph string) string {
	if fsm == nil {
		return graph
	}

	for name, state := range fsm.States {
		for _, transition := range state.Transition {
			graph += "\n" + "cluster_" + name.String() + " -> cluster_" + transition.String()
		}

		graph += "\nsubgraph cluster_" + name.String() + "{\n"
		if (state.Locked) {
			graph += "style=\"dotted\"\n"
		} else {
			graph += "style=\"\"\n"
		}
		if (fsm.Current == name) {
			graph += "color=\"blue\"\n"
			graph += "fontcolor=\"blue\"\n"
		} else {
			graph += "color=\"\"\n"
			graph += "fontcolor=\"\"\n"
		}
		graph += "label=" + name.String()
		graph = getGraphviz(state.Sub, graph)
		graph += "\n}"
	}
	return graph
}

func (eduvpn *VPNState) GenerateGraph() string {
	graph := "digraph fsm {\n"
	graph += "nodesep=2"
	graph = getGraphviz(eduvpn.FSM, graph)
	graph += "\n}"

	return graph
}

func (eduvpn *VPNState) InitializeFSM() {
	// The states when a server is authenticated
	serverAuthenticated := &FSMState{Sub: &FSM{States: FSMStates{
		SERVER_DISCONNECTED: {Transition: FSMTransitions{SERVER_CONNECTED}},
		SERVER_CONNECTED:    {Transition: FSMTransitions{SERVER_DISCONNECTED}},
	}, Current: SERVER_DISCONNECTED}, Transition: FSMTransitions{SERVER_NOT_AUTHENTICATED}}

	// The states when a server is not authenticated
	serverNotAuthenticated := &FSMState{Sub: &FSM{States: FSMStates{
		// In this state we cannot exit to the parent state
		// As the parent state can go to authenticated
		SERVER_INITIALIZED: {Transition: FSMTransitions{SERVER_OAUTH_STARTED}, Locked: true},

		// The state that indicates oauth is in progress
		SERVER_OAUTH_STARTED:  {Transition: FSMTransitions{SERVER_OAUTH_FINISHED}, Locked: true},
		SERVER_OAUTH_FINISHED: {Transition: FSMTransitions{SERVER_OAUTH_STARTED}},
	}, Current: SERVER_INITIALIZED}, Transition: FSMTransitions{SERVER_AUTHENTICATED}}

	// The states of the server, it has authenticated and not authenticated ass sub states
	serverStates := &FSMState{Sub: &FSM{States: FSMStates{
		SERVER_AUTHENTICATED:     serverAuthenticated,
		SERVER_NOT_AUTHENTICATED: serverNotAuthenticated,
	}, Current: SERVER_NOT_AUTHENTICATED}, Transition: FSMTransitions{CONFIG_NOSERVER}}

	// The state when a server is registered
	registeredState := &FSMState{Sub: &FSM{States: FSMStates{
		// When no server has been chosen, we have no sub states
		CONFIG_NOSERVER: {Transition: FSMTransitions{CONFIG_CHOSENSERVER}},
		// A server has been chosen, it has substates such as oauth, connected and disconnected
		CONFIG_CHOSENSERVER: serverStates,
	}, Current: CONFIG_NOSERVER}, Transition: FSMTransitions{APP_DEREGISTERED}}

	deregisteredState := &FSMState{Transition: FSMTransitions{APP_REGISTERED}}

	eduvpn.FSM = &FSM{
		States: FSMStates{
			APP_REGISTERED: registeredState, APP_DEREGISTERED: deregisteredState,
		}, Current: APP_DEREGISTERED,
	}
}
