package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
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
func openBrowser(data interface{}) {
	str, ok := data.(string)
	if !ok {
		return
	}
	// double check URL scheme
	u, err := url.Parse(str)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed parsing url", err)
		return
	}
	// Double check the scheme
	if u.Scheme != "https" {
		fmt.Fprintln(os.Stderr, "got invalid scheme for URL:", u.String())
		return
	}
	fmt.Println("Please open your browser with URL:", u.String())
	// In practice, a client should open the browser here
	// But be careful with which commands you execute with this input
	// As a client you should do enough input validation such that opening the browser does not have unwanted side effects
	// We do our best to validate the URL in this example by parsing if it's a URL and additionally failing if the scheme is not HTTPS
	// Note that the library already tries it best to validate data from the server, but a client should always be careful which data it uses
}

// Ask for a profile in the command line.
func sendProfile(state *client.Client, data interface{}) {
	fmt.Printf("Multiple VPN profiles found. Please select a profile by entering e.g. 1")
	sps, ok := data.(*server.ProfileInfo)
	if !ok {
		fmt.Fprintln(os.Stderr, "invalid data type")
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
		fmt.Fprintln(os.Stderr, "invalid profile chosen, please retry")
		sendProfile(state, data)
		return
	}

	p := sps.Info.ProfileList[idx-1]
	fmt.Println("Sending profile ID", p.ID)
	if err := state.SetProfileID(p.ID); err != nil {
		fmt.Fprintln(os.Stderr, "failed setting profile with error", err)
	}
}

// The callback function
// If OAuth is started we open the browser with the Auth URL
// If we ask for a profile, we send the profile using command line input
// Note that this has an additional argument, the vpn state which was wrapped into this callback function below.
func stateCallback(state *client.Client, _ client.FSMStateID, newState client.FSMStateID, data interface{}) {
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
		"1.1.2-cli",
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
		fmt.Fprintf(os.Stderr, "Error getting config: %s\nCause:\n%s\nStack trace:\n%s\n\n'",
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
