// Package fsm defines a finite state machine and has the ability to save this state machine to a graph file
// This graph file can be visualized using mermaid.js
package fsm

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
	"github.com/eduvpn/eduvpn-common/types"
)

type (
	//StateID represents the Identifier of the state
	FSMStateID      int8
	//StateIDSlice represents the list of state identifiers
	FSMStateIDSlice []FSMStateID
)

func (v FSMStateIDSlice) Len() int {
	return len(v)
}

func (v FSMStateIDSlice) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v FSMStateIDSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Transition indicates an arrow in the state graph
type FSMTransition struct {
	// To represents the to-be-new state
	To          FSMStateID
	// Description is what type of message the arrow gets in the graph
	Description string
}

type (
	FSMStates map[FSMStateID]FSMState
)

// State represents a single node in the graph
type FSMState struct {
	// Transitions indicates which out arrows this node has
	Transitions []FSMTransition
}

// FSM represents the total graph
type FSM struct {
	// States is the map from state ID to states
	States  FSMStates

	// Current is the current state represented by the identifier
	Current FSMStateID

	// Name represents the descriptive name of this state machine
	Name          string

	// StateCallback is the function ran when a transition occurs
	// It takes the old state, the new state and the data and returns if this is handled by the client
	StateCallback func(FSMStateID, FSMStateID, interface{}) bool

	// Directory represents the path where the state graph is stored
	Directory     string

	// Generate represents whether we want to generate the graph
	Generate         bool

	// GetStateName gets the name of a state as a string
	GetStateName       func(FSMStateID) string
}

// Init initializes the state machine and sets it to the given current state
func (fsm *FSM) Init(
	current FSMStateID,
	states map[FSMStateID]FSMState,
	callback func(FSMStateID, FSMStateID, interface{}) bool,
	directory string,
	nameGen func(FSMStateID) string,
	generate bool,
) {
	fsm.States = states
	fsm.Current = current
	fsm.StateCallback = callback
	fsm.Directory = directory
	fsm.GetStateName = nameGen
	fsm.Generate = generate
}

// InState returns whether or not the state machine is in the given 'check' state
func (fsm *FSM) InState(check FSMStateID) bool {
	return check == fsm.Current
}

// HasTransition checks whether or not the state machine has a transition to the given 'check' state
func (fsm *FSM) HasTransition(check FSMStateID) bool {
	for _, transitionState := range fsm.States[fsm.Current].Transitions {
		if transitionState.To == check {
			return true
		}
	}

	return false
}

// getGraphFilename gets the full path to the graph filename including the .graph extension
func (fsm *FSM) getGraphFilename(extension string) string {
	debugPath := path.Join(fsm.Directory, "graph")
	return fmt.Sprintf("%s%s", debugPath, extension)
}

// writeGraph writes the state machine to a .graph file
func (fsm *FSM) writeGraph() {
	graph := fsm.GenerateGraph()
	graphFile := fsm.getGraphFilename(".graph")
	graphImgFile := fsm.getGraphFilename(".png")
	f, err := os.Create(graphFile)
	if err != nil {
		return
	}

	_, writeErr := f.WriteString(graph)
	f.Close()
	if writeErr != nil {
		cmd := exec.Command("mmdc", "-i", graphFile, "-o", graphImgFile, "--scale", "4")
		// Generating is best effort
		_ = cmd.Start()
	}
}

// GoTransitionRequired transitions the state machine to a new state with associated state data 'data'
// If this transition is not handled by the client, it returns an error
func (fsm *FSM) GoTransitionRequired(newState FSMStateID, data interface{}) error {
	oldState := fsm.Current
	if !fsm.GoTransitionWithData(newState, data) {
		return types.NewWrappedError("failed required transition", fmt.Errorf("required transition not handled, from: %s -> to: %s", fsm.GetStateName(oldState), fsm.GetStateName(newState)))
	}
	return nil
}

// GoTransitionWithData is a helper that transitions the state machine toward the 'newState' with associated state data 'data'
// It returns whether or not the transition is handled by the client
func (fsm *FSM) GoTransitionWithData(newState FSMStateID, data interface{}) bool {
	ok := fsm.HasTransition(newState)

	handled := false
	if ok {
		oldState := fsm.Current
		fsm.Current = newState
		if fsm.Generate {
			fsm.writeGraph()
		}

		handled = fsm.StateCallback(oldState, newState, data)
	}

	return handled
}

// GoTransition is an alias to call GoTransitionWithData but have an empty string as data
func (fsm *FSM) GoTransition(newState FSMStateID) bool {
	// No data means the callback is never required
	return fsm.GoTransitionWithData(newState, "")
}

// generateMermaidGraph generates a graph suitable to be converted by the mermaid.js tool
// it returns the graph as a string
func (fsm *FSM) generateMermaidGraph() string {
	graph := "graph TD\n"
	sortedFSM := make(FSMStateIDSlice, 0, len(fsm.States))
	for stateID := range fsm.States {
		sortedFSM = append(sortedFSM, stateID)
	}
	sort.Sort(sortedFSM)
	for _, state := range sortedFSM {
		transitions := fsm.States[state].Transitions
		for _, transition := range transitions {
			if state == fsm.Current {
				graph += "\nstyle " + fsm.GetStateName(state) + " fill:cyan\n"
			} else {
				graph += "\nstyle " + fsm.GetStateName(state) + " fill:white\n"
			}
			graph += fsm.GetStateName(
				state,
			) + "(" + fsm.GetStateName(
				state,
			) + ") " + "-->|" + transition.Description + "| " + fsm.GetStateName(
				transition.To,
			) + "\n"
		}
	}
	return graph
}

// GenerateGraph generates a mermaid graph if the state machine is initialized
// If the graph cannot be generated, it returns the empty string
func (fsm *FSM) GenerateGraph() string {
	if fsm.GetStateName != nil {
		return fsm.generateMermaidGraph()
	}

	return ""
}
