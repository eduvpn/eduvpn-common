// Package fsm defines a finite state machine
package fsm

import "fmt"

// State represents a single node in the graph.
type State struct {
	// Transitions indicates which out arrows this node has
	Transitions []Transition
}

type (
	// StateID represents the Identifier of the state.
	StateID int8
	// StateIDSlice represents the list of state identifiers.
	StateIDSlice []StateID
	// States is the map from state identifier to the state itself
	States map[StateID]State
)

// Len is defined here such that we can sort the slice
func (v StateIDSlice) Len() int {
	return len(v)
}

// Less is defined here such that we can sort the slice
func (v StateIDSlice) Less(i, j int) bool {
	return v[i] < v[j]
}

// Swap is defined here such that we can sort the slice
func (v StateIDSlice) Swap(i, j int) {
	v[i], v[j] = v[j], v[i]
}

// Transition indicates an arrow in the state graph.
type Transition struct {
	// To represents the to-be-new state
	To StateID
	// Description is what type of message the arrow gets in the graph
	Description string
}

// FSM represents the total graph.
type FSM struct {
	// States is the map from state ID to states
	States States

	// Current is the current state represented by the identifier
	Current StateID

	// Name represents the descriptive name of this state machine
	Name string

	// StateCallback is the function ran when a transition occurs
	// It takes the old state, the new state and the data and returns if this is handled by the client
	StateCallback func(StateID, StateID, interface{}) bool

	// GetStateName gets the name of a state as a string
	GetStateName func(StateID) string

	// initial is the initial state that we can always go back to
	initial StateID
}

// NewFSM creates a new finite state machine
func NewFSM(current StateID, states States, callback func(StateID, StateID, interface{}) bool, nameGen func(StateID) string) FSM {
	return FSM{
		States:        states,
		Current:       current,
		StateCallback: callback,
		GetStateName:  nameGen,
		initial:       current,
	}
}

// InState returns whether or not the state machine is in the given 'check' state.
func (fsm *FSM) InState(check StateID) bool {
	return check == fsm.Current
}

// CheckTransition returns an error whether or not a transition to
// state `desired` is possible
func (fsm *FSM) CheckTransition(desired StateID) error {
	// initial or begin state is fine
	// 0 = deregistered
	if desired == fsm.initial || desired == 0 {
		return nil
	}
	for _, ts := range fsm.States[fsm.Current].Transitions {
		if ts.To == desired {
			return nil
		}
	}
	return fmt.Errorf("fsm invalid transition attempt from '%s' to '%s'", fsm.GetStateName(fsm.Current), fsm.GetStateName(desired))
}

// GoTransitionRequired transitions the state machine to a new state with associated state data 'data'
// If this transition is not handled by the client, it returns an error.
func (fsm *FSM) GoTransitionRequired(newState StateID, data interface{}) error {
	oldState := fsm.Current

	handled, err := fsm.GoTransitionWithData(newState, data)
	// transition ios not possible
	if err != nil {
		return err
	}
	// transition is not handled
	if !handled {
		return fmt.Errorf("fsm failed transition from '%s' to '%s', is this required transition handled?", fsm.GetStateName(oldState), fsm.GetStateName(newState))
	}
	return nil
}

// GoTransitionWithData is a helper that transitions the state machine toward the 'newState' with associated state data 'data'
// It returns whether or not the transition is handled by the client.
func (fsm *FSM) GoTransitionWithData(newState StateID, data interface{}) (bool, error) {
	if err := fsm.CheckTransition(newState); err != nil {
		return false, err
	}

	prev := fsm.Current
	fsm.Current = newState
	return fsm.StateCallback(prev, newState, data), nil
}

// GoTransition is an alias to call GoTransitionWithData but have an empty string as data.
func (fsm *FSM) GoTransition(newState StateID) (bool, error) {
	// No data means the callback is never required
	return fsm.GoTransitionWithData(newState, "")
}
