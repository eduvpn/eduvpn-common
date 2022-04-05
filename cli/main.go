package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	eduvpn "github.com/jwijenbergh/eduvpn-common/src"
)

func openBrowser(urlString string) {
	fmt.Printf("OAuth: Initialized with AuthURL %s\n", urlString)
	fmt.Println("OAuth: Opening browser with xdg-open...")
	exec.Command("xdg-open", urlString).Start()
}

func logState(oldState string, newState string, data string) {
	log.Printf("State: %s -> State: %s with data %s\n", oldState, newState, data)

	if newState == "SERVER_OAUTH_STARTED" {
		openBrowser(data)
	}
}

func getGraphviz(fsm *eduvpn.FSM, graph string) string {
	if fsm == nil {
		return graph
	}

	for name, state := range fsm.States {
		for _, transition := range state.Transition {
			graph += "\n" + "cluster_" + name.String() + "-> cluster_" + transition.String()
		}

		graph += "\nsubgraph cluster_" + name.String() + "{\n"
		if (state.Locked) {
			graph += "bgcolor=\"red\"\n"
		}
		if (fsm.Current == name) {
			graph += "style=\"bold\"\n"
			graph += "color=\"blue\"\n"
		} else {
			graph += "style=\"\"\n"
			graph += "color=\"\"\n"
		}
		graph += "label=" + name.String()
		graph = getGraphviz(state.Sub, graph)
		graph += "\n}"
	}
	return graph
}

func generateGraph() string {
	state := eduvpn.GetVPNState()

	state.InitializeFSM()


	graph := "digraph fsm {\n"
	graph += "nodesep=2"
	graph = getGraphviz(state.FSM, graph)
	graph += "\n}"

	return graph
}

func main() {
	fileGraph := flag.String("dumpgraph", "", "Dump the FSM to a graphviz fdp file")
	urlArg := flag.String("url", "", "The url of the vpn")
	flag.Parse()

	fileGraphString := *fileGraph
	if fileGraphString != "" {
		f, err := os.Create(fileGraphString)

		if err != nil {
			log.Fatalf("Failed to create file %s with error %v", fileGraphString, err)
		}

		defer f.Close()

		f.WriteString(generateGraph())

		log.Printf("Graph written to file: %s, use 'fdp %s -Tsvg > graph.svg' from graphviz to save to a svg file called graph.svg\n", fileGraphString, fileGraphString)
		return
	}
	urlString := *urlArg

	if urlString != "" {
		if !strings.HasPrefix(urlString, "https://") {
			urlString = "https://" + urlString
		}

		state := eduvpn.GetVPNState()

		state.Register("org.eduvpn.app.linux", "configs", logState, true)
		config, configErr := state.Connect(urlString)

		if configErr != nil {
			fmt.Printf("Config error %v", configErr)
			return
		}

		fmt.Println(config)

		return
	}

	flag.PrintDefaults()
}
