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

	state := eduvpn.Register("org.eduvpn.app.linux", urlString)
	authURL, err := eduvpn.InitializeOAuth(state)
	if err != nil {
		log.Fatal(err)
	}
	openBrowser(authURL)
	oauthErr := eduvpn.FinishOAuth(state)
	if oauthErr != nil {
		log.Fatal(oauthErr)
	}
	infoString, infoErr := eduvpn.APIAuthenticatedInfo(state)
	if infoErr != nil {
		log.Fatal(infoErr)
	}
	log.Println(infoString)
}
