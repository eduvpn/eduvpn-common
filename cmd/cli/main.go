package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	eduvpn "github.com/jwijenbergh/eduvpn-common"
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

func main() {
	urlArg := flag.String("url", "", "The url of the vpn")
	flag.Parse()

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

		state.Deregister()

		return
	}

	flag.PrintDefaults()
}
