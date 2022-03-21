package main

import (
	"flag"
	eduvpn "github.com/jwijenbergh/eduvpn-common/src"
	"log"
	"os/exec"
	"strings"
)

func openBrowser(urlString string) {
	log.Printf("OAuth: Initialized with AuthURL %s\n", urlString)
	log.Println("OAuth: Opening browser with xdg-open...")
	exec.Command("xdg-open", urlString).Start()
}

func logState(oldState string, newState string) {
	log.Printf("State: %s -> State: %s\n", oldState, newState)
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

	eduvpn.Register(state, "org.eduvpn.app.linux", logState)
	state.Server = &eduvpn.Server{}
	serverInitializeErr := state.Server.Initialize(urlString)
	if serverInitializeErr != nil {
		log.Fatal(serverInitializeErr)
	}

	authURL, err := state.InitializeOAuth()
	if err != nil {
		log.Fatal(err)
	}
	openBrowser(authURL)
	oauthErr := state.FinishOAuth()
	if oauthErr != nil {
		log.Fatal(oauthErr)
	}
	wireguardKey, wireguardErr := eduvpn.WireguardGenerateKey()

	if wireguardErr != nil {
		log.Fatal(wireguardErr)
	}
	configString, configErr := state.APIConnectWireguard(wireguardKey.PublicKey().String())
	if configErr != nil {
		log.Fatal(configErr)
	}
	log.Println(eduvpn.WireguardConfigAddKey(configString, wireguardKey))
}
