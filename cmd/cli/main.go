package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eduvpn/eduvpn-common/client"
	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/go-errors/errors"
)

type ServerTypes int8

const (
	ServerTypeInstituteAccess ServerTypes = iota
	ServerTypeSecureInternet
	ServerTypeCustom
)

// Open a browser with xdg-open.
func openBrowser(url interface{}) {
	str, ok := url.(string)
	if !ok {
		return
	}
	fmt.Printf("OAuth: Initialized with AuthURL %s\n", str)
	fmt.Println("OAuth: Opening browser with xdg-open...")
	if exec.Command("xdg-open", str).Start() != nil {
		// TODO(): Shouldn't this if statement be inverted?
		fmt.Println("OAuth: Browser opened with xdg-open...")
	}
}

// Ask for a profile in the command line.
func sendProfile(state *client.Client, data interface{}) {
	fmt.Printf("Multiple VPN profiles found. Please select a profile by entering e.g. 1")
	sps, ok := data.(*server.ProfileInfo)
	if !ok {
		fmt.Println("Invalid data type")
		return
	}

	ps := ""
	for i, p := range sps.Info.ProfileList {
		ps += fmt.Sprintf("\n%d - %s", i+1, p.DisplayName)
	}

	// Show the profiles
	fmt.Println(ps)

	var idx int
	if _, err := fmt.Scanf("%d", &idx); err != nil || idx <= 0 ||
		idx > len(sps.Info.ProfileList) {
		fmt.Println("invalid profile chosen, please retry")
		sendProfile(state, data)
		return
	}

	p := sps.Info.ProfileList[idx-1]
	fmt.Println("Sending profile ID", p.ID)
	if err := state.SetProfileID(p.ID); err != nil {
		fmt.Println("Failed setting profile with error", err)
	}
}

// The callback function
// If OAuth is started we open the browser with the Auth URL
// If we ask for a profile, we send the profile using command line input
// Note that this has an additional argument, the vpn state which was wrapped into this callback function below.
func stateCallback(state *client.Client, oldState client.FSMStateID, newState client.FSMStateID, data interface{}) {
	if newState == client.StateOAuthStarted {
		openBrowser(data)
	}

	if newState == client.StateAskProfile {
		sendProfile(state, data)
	}
}

// Get a config for Institute Access or Secure Internet Server.
func getConfig(state *client.Client, url string, srvType ServerTypes) (*client.ConfigData, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	// Prefer TCP is set to False
	if srvType == ServerTypeInstituteAccess {
		_, err := state.AddInstituteServer(url)
		if err != nil {
			return nil, err
		}
		return state.GetConfigInstituteAccess(url, false, oauth.Token{})
	} else if srvType == ServerTypeCustom {
		_, err := state.AddCustomServer(url)
		if err != nil {
			return nil, err
		}
		return state.GetConfigCustomServer(url, false, oauth.Token{})
	}
	_, err := state.AddSecureInternetHomeServer(url)
	if err != nil {
		return nil, err
	}
	return state.GetConfigSecureInternet(url, false, oauth.Token{})
}

// Get a config for a single server, Institute Access or Secure Internet.
func printConfig(url string, srvType ServerTypes) {
	c := &client.Client{}

	err := c.Register(
		"org.eduvpn.app.linux",
		"configs",
		"en",
		func(old client.FSMStateID, new client.FSMStateID, data interface{}) bool {
			stateCallback(c, old, new, data)
			return true
		},
		true,
	)
	if err != nil {
		fmt.Printf("Register error: %v", err)
		return
	}

	defer c.Deregister()

	cfg, err := getConfig(c, url, srvType)
	if err != nil {
		err1 := err.(*errors.Error)
		// Show the usage of tracebacks and causes
		fmt.Printf("Error getting config: %s\nCause:\n%s\nStack trace:\n%s\n\n'",
			err1.Error(), err1.Err, err1.ErrorStack())
		return
	}

	fmt.Println("Obtained config:", cfg.Config)
}

// The main function
// It parses the arguments and executes the correct functions.
func main() {
	cu := flag.String("get-custom", "", "The url of a custom server to connect to")
	u := flag.String("get-institute", "", "The url of an institute to connect to")
	sec := flag.String("get-secure", "", "Gets secure internet servers")
	flag.Parse()

	// Connect to a VPN by getting an Institute Access config
	switch {
	case *cu != "":
		printConfig(*cu, ServerTypeCustom)
	case *u != "":
		printConfig(*u, ServerTypeInstituteAccess)
	case *sec != "":
		printConfig(*sec, ServerTypeSecureInternet)
	default:
		flag.PrintDefaults()
	}
}
