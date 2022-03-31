package main

import (
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strings"

	eduvpn "github.com/jwijenbergh/eduvpn-common/src"
)

func openBrowser(urlString string) {
	log.Printf("OAuth: Initialized with AuthURL %s\n", urlString)
	log.Println("OAuth: Opening browser with xdg-open...")
	exec.Command("xdg-open", urlString).Start()
}

func logState(oldState string, newState string, data string) {
	log.Printf("State: %s -> State: %s with data %s\n", oldState, newState, data)

	if newState == "SERVER_OAUTH_STARTED" {
		openBrowser(data)
	}
}

func main() {
	urlArg := flag.String("url", "", "The url of the vpn")
	flag.Parse()

	urlString := *urlArg

	if urlString == "" {
		log.Fatal("Error: -url is required")
	}

	if !strings.HasPrefix(urlString, "https://") {
		urlString = "https://" + urlString
	}

	state := eduvpn.GetVPNState()

	state.Register("org.eduvpn.app.linux", "configs", logState)
	config, configErr := state.Connect(urlString)

	if configErr != nil {
		fmt.Printf("Config error %v", configErr)
		return
	}

	print(config)
}
