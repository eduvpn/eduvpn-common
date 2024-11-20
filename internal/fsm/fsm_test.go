package fsm

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
)

func TestSortSlice(t *testing.T) {
	cases := []struct {
		In   StateIDSlice
		Want StateIDSlice
	}{
		{
			In:   StateIDSlice{0, 1, 2, 3},
			Want: StateIDSlice{0, 1, 2, 3},
		},
		{
			In:   StateIDSlice{0, 3, 2, 3},
			Want: StateIDSlice{0, 2, 3, 3},
		},
		{
			In:   StateIDSlice{0, 3, 2, 1},
			Want: StateIDSlice{0, 1, 2, 3},
		},
	}

	for _, c := range cases {
		sort.Sort(c.In)
		if !reflect.DeepEqual(c.In, c.Want) {
			t.Fatalf("slice not sorted, in: %v, want: %v", c.In, c.Want)
		}
	}
}

func testFSM() FSM {
	states := States{
		0: State{
			Transitions: []Transition{
				{To: 1},
				{To: 2},
			},
		},
		1: State{
			Transitions: []Transition{},
		},
		2: State{
			Transitions: []Transition{
				{To: 1},
			},
		},
		// isolated state
		3: State{
			Transitions: []Transition{},
		},
	}
	cb := func(StateID, StateID, interface{}) bool {
		return false
	}
	namecb := func(in StateID) string {
		return fmt.Sprintf("Test%d", in)
	}
	return NewFSM(0, states, cb, namecb)
}

func TestCheckTransition(t *testing.T) {
	machine := testFSM()

	cases := []struct {
		In      StateID
		WantErr string
	}{
		{
			In:      1,
			WantErr: "",
		},
		{
			In:      2,
			WantErr: "fsm invalid transition attempt from 'Test1' to 'Test2'",
		},
		// we can always go back to the initial state
		{
			In:      0,
			WantErr: "",
		},
		{
			In:      2,
			WantErr: "",
		},
		{
			In:      3,
			WantErr: "fsm invalid transition attempt from 'Test2' to 'Test3'",
		},
	}

	for _, c := range cases {
		err := machine.CheckTransition(c.In)
		test.AssertError(t, err, c.WantErr)

		// mock a transition
		if err == nil {
			machine.Current = c.In
		}
	}
}

func TestGoTransitionRequired(t *testing.T) {
	machine := testFSM()

	cases := []struct {
		In      StateID
		Handle  bool
		WantErr string
		Data    string
	}{
		{
			In:      1,
			WantErr: "fsm failed transition from 'Test0' to 'Test1', is this required transition handled?",
		},
		{
			In:      1,
			Handle:  true,
			WantErr: "",
		},
		{
			In:      2,
			WantErr: "fsm invalid transition attempt from 'Test1' to 'Test2'",
		},
		// we can always go back to the initial state
		{
			In:      0,
			Handle:  true,
			WantErr: "",
			Data:    "test",
		},
		{
			In:      2,
			Handle:  false,
			WantErr: "fsm failed transition from 'Test0' to 'Test2', is this required transition handled?",
		},
		{
			In:      3,
			WantErr: "fsm invalid transition attempt from 'Test0' to 'Test3'",
		},
	}

	for _, c := range cases {
		curr := machine.Current
		machine.StateCallback = func(_ StateID, newState StateID, data interface{}) bool {
			if c.WantErr == "" && newState != c.In {
				t.Fatalf("new state is not what we want, got: %v, want: %v", newState, c.In)
			}
			v, ok := data.(string)
			if !ok {
				t.Fatal("data is not a string")
			}
			if c.Data != v {
				t.Fatalf("data is not equal, got: %v, want: %v", v, c.Data)
			}
			return c.Handle
		}
		err := machine.GoTransitionRequired(c.In, c.Data)
		test.AssertError(t, err, c.WantErr)

		// mock setting state back if the handle was not successful
		if c.WantErr != "" {
			machine.Current = curr
		}
	}
}

func TestGoTransition(t *testing.T) {
	machine := testFSM()

	cases := []struct {
		In      StateID
		Handle  bool
		Data    string
		WantErr string
	}{
		{
			In:      1,
			Data:    "test",
			WantErr: "",
		},
		// self-loops not allowed
		{
			In:      1,
			WantErr: "fsm invalid transition attempt from 'Test1' to 'Test1'",
		},
		{
			In:      2,
			WantErr: "fsm invalid transition attempt from 'Test1' to 'Test2'",
		},
		// we can always go back to the initial state
		{
			In:      0,
			Handle:  true,
			WantErr: "",
		},
		{
			In:      2,
			Handle:  false,
			WantErr: "",
		},
		{
			In:      3,
			WantErr: "fsm invalid transition attempt from 'Test2' to 'Test3'",
		},
	}

	for _, c := range cases {
		curr := machine.Current
		machine.StateCallback = func(StateID, StateID, interface{}) bool {
			return c.Handle
		}
		machine.StateCallback = func(_ StateID, newState StateID, data interface{}) bool {
			if c.WantErr == "" && newState != c.In {
				t.Fatalf("new state is not what we want, got: %v, want: %v", newState, c.In)
			}
			v, ok := data.(string)
			if !ok {
				t.Fatal("data is not a string")
			}
			if c.Data != v {
				t.Fatalf("data is not equal, got: %v, want: %v", v, c.Data)
			}
			return c.Handle
		}
		var ghandle bool
		var err error
		if c.Data != "" {
			ghandle, err = machine.GoTransitionWithData(c.In, c.Data)
		} else {
			ghandle, err = machine.GoTransition(c.In)
		}
		test.AssertError(t, err, c.WantErr)

		if ghandle != c.Handle {
			t.Fatalf("handled bool not equal, got: %v, want: %v", ghandle, c.Handle)
		}

		// mock setting state back if the handle was not successful
		if c.WantErr != "" {
			machine.Current = curr
		} else if !machine.InState(c.In) {
			t.Fatalf("after successful transition, FSM is not in state: %v", c.In)
		}
	}
}
