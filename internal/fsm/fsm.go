package fsm

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"sort"
)

type (
	FSMStateID      int8
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

type FSMTransition struct {
	To          FSMStateID
	Description string
}

type (
	FSMStates map[FSMStateID]FSMState
)

type FSMState struct {
	Transitions []FSMTransition

	// Which state to go back to on a back transition
	BackState FSMStateID
}

type FSM struct {
	States  FSMStates
	Current FSMStateID

	// Info to be passed from the parent state
	Name          string
	StateCallback func(FSMStateID, FSMStateID, interface{})
	Directory     string
	Debug         bool
	GetName       func(FSMStateID) string
}

func (fsm *FSM) Init(
	name string,
	current FSMStateID,
	states map[FSMStateID]FSMState,
	callback func(FSMStateID, FSMStateID, interface{}),
	directory string,
	nameGen func(FSMStateID) string,
	debug bool,
) {
	fsm.States = states
	fsm.Current = current
	fsm.Name = name
	fsm.StateCallback = callback
	fsm.Directory = directory
	fsm.GetName = nameGen
	fsm.Debug = debug
}

func (fsm *FSM) InState(check FSMStateID) bool {
	return check == fsm.Current
}

func (fsm *FSM) HasTransition(check FSMStateID) bool {
	for _, transition_state := range fsm.States[fsm.Current].Transitions {
		if transition_state.To == check {
			return true
		}
	}

	return false
}

func (fsm *FSM) getGraphFilename(extension string) string {
	debugPath := path.Join(fsm.Directory, fsm.Name)
	return fmt.Sprintf("%s%s", debugPath, extension)
}

func (fsm *FSM) writeGraph() {
	graph := fsm.GenerateGraph()
	graphFile := fsm.getGraphFilename(".graph")
	graphImgFile := fsm.getGraphFilename(".png")
	f, err := os.Create(graphFile)
	if err != nil {
		return
	}

	f.WriteString(graph)
	f.Close()
	cmd := exec.Command("mmdc", "-i", graphFile, "-o", graphImgFile, "--scale", "4")

	cmd.Start()
}

func (fsm *FSM) GoBack() {
	fsm.GoTransition(fsm.States[fsm.Current].BackState)
}

func (fsm *FSM) GoTransitionWithData(newState FSMStateID, data interface{}, background bool) bool {
	ok := fsm.HasTransition(newState)

	if ok {
		oldState := fsm.Current
		fsm.Current = newState
		if fsm.Debug {
			fsm.writeGraph()
		}

		if background {
			go fsm.StateCallback(oldState, newState, data)
		} else {
			fsm.StateCallback(oldState, newState, data)
		}
	}

	return ok
}

func (fsm *FSM) GoTransition(newState FSMStateID) bool {
	return fsm.GoTransitionWithData(newState, "{}", false)
}

func (fsm *FSM) generateMermaidGraph() string {
	graph := "graph TD\n"
	sorted_fsm := make(FSMStateIDSlice, 0, len(fsm.States))
	for state_id := range fsm.States {
		sorted_fsm = append(sorted_fsm, state_id)
	}
	sort.Sort(sorted_fsm)
	for _, state := range sorted_fsm {
		transitions := fsm.States[state].Transitions
		for _, transition := range transitions {
			if state == fsm.Current {
				graph += "\nstyle " + fsm.GetName(state) + " fill:cyan\n"
			} else {
				graph += "\nstyle " + fsm.GetName(state) + " fill:white\n"
			}
			graph += fsm.GetName(
				state,
			) + "(" + fsm.GetName(
				state,
			) + ") " + "-->|" + transition.Description + "| " + fsm.GetName(
				transition.To,
			) + "\n"
		}
	}
	return graph
}

func (fsm *FSM) GenerateGraph() string {
	if fsm.GetName != nil {
		return fsm.generateMermaidGraph()
	}

	return ""
}
