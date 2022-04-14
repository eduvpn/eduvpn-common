package main

import (
	"flag"
	"fmt"
	"errors"
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
	fmt.Printf("State: %s -> State: %s with data %s\n", oldState, newState, data)

	if newState == "OAuth_Started" {
		openBrowser(data)
	}
}

func writeGraph(filename string) error {
	state := eduvpn.GetVPNState()

	state.InitializeFSM()

	graph := state.GenerateGraph()

	f, err := os.Create(filename)

	if err != nil {
		return errors.New(fmt.Sprintf("Failed to create file %s with error %v", filename, err))
	}

	defer f.Close()

	f.WriteString(graph)

	fmt.Printf("Graph written to file: %s, use 'fdp %s -Tsvg > graph.svg' from graphviz to save to a svg file called graph.svg\n", filename, filename)

	return nil
}

func main() {
	fileGraph := flag.String("dumpgraph", "", "Dump the FSM to a graphviz fdp file")
	urlArg := flag.String("url", "", "The url of the vpn")
	flag.Parse()

	fileGraphString := *fileGraph
	if fileGraphString != "" {
		writeGraph(fileGraphString)
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
