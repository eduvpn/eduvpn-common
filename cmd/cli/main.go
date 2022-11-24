package main

import (
	"flag"
	"fmt"
	"os/exec"
	"strings"

	"github.com/eduvpn/eduvpn-common/client"
	"github.com/eduvpn/eduvpn-common/types"
	"github.com/eduvpn/eduvpn-common/internal/server"
)

type ServerTypes int8

const (
	ServerTypeInstituteAccess ServerTypes = iota
	ServerTypeSecureInternet
	ServerTypeCustom
)

// Open a browser with xdg-open
func openBrowser(url interface{}) {
	urlString, ok := url.(string)

	if !ok {
		return
	}
	fmt.Printf("OAuth: Initialized with AuthURL %s\n", urlString)
	fmt.Println("OAuth: Opening browser with xdg-open...")
	cmdErr := exec.Command("xdg-open", urlString).Start()
	if cmdErr != nil {
		fmt.Println("OAuth: Browser opened with xdg-open...")
	}
}

// Ask for a profile in the command line
func sendProfile(state *client.Client, data interface{}) {
	fmt.Printf("Multiple VPN profiles found. Please select a profile by entering e.g. 1")
	serverProfiles, ok := data.(*server.ServerProfileInfo)

	if !ok {
		fmt.Println("Invalid data type")
		return
	}

	var profiles string

	for index, profile := range serverProfiles.Info.ProfileList {
		profiles += fmt.Sprintf("\n%d - %s", index+1, profile.DisplayName)
	}

	// Show the profiles
	fmt.Println(profiles)

	var chosenProfile int
	_, scanErr := fmt.Scanf("%d", &chosenProfile)

	if scanErr != nil || chosenProfile <= 0 ||
		chosenProfile > len(serverProfiles.Info.ProfileList) {
		fmt.Println("invalid profile chosen, please retry")
		sendProfile(state, data)
		return
	}

	profile := serverProfiles.Info.ProfileList[chosenProfile-1]
	fmt.Println("Sending profile ID", profile.ID)
	profileErr := state.SetProfileID(profile.ID)

	if profileErr != nil {
		fmt.Println("Failed setting profile with error", profileErr)
	}
}

// The callback function
// If OAuth is started we open the browser with the Auth URL
// If we ask for a profile, we send the profile using command line input
// Note that this has an additional argument, the vpn state which was wrapped into this callback function below
func stateCallback(
	state *client.Client,
	oldState client.FSMStateID,
	newState client.FSMStateID,
	data interface{},
) {
	if newState == client.StateOAuthStarted {
		openBrowser(data)
	}

	if newState == client.StateAskProfile {
		sendProfile(state, data)
	}
}

// Get a config for Institute Access or Secure Internet Server
func getConfig(state *client.Client, url string, serverType ServerTypes) (string, string, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	// Prefer TCP is set to False
	if serverType == ServerTypeInstituteAccess {
		_, addErr := state.AddInstituteServer(url)
		if addErr != nil {
			return "", "", addErr
		}
		return state.GetConfigInstituteAccess(url, false)
	} else if serverType == ServerTypeCustom {
		_, addErr := state.AddCustomServer(url)
		if addErr != nil {
			return "", "", addErr
		}
		return state.GetConfigCustomServer(url, false)
	}
	_, addErr := state.AddSecureInternetHomeServer(url)
	if addErr != nil {
		return "", "", addErr
	}
	return state.GetConfigSecureInternet(url, false)
}

// Get a config for a single server, Institute Access or Secure Internet
func printConfig(url string, serverType ServerTypes) {
	state := &client.Client{}

	registerErr := state.Register(
		"org.eduvpn.app.linux",
		"configs",
		"en",
		func(old client.FSMStateID, new client.FSMStateID, data interface{}) bool {
			stateCallback(state, old, new, data)
			return true
		},
		true,
	)
	if registerErr != nil {
		fmt.Printf("Register error: %v", registerErr)
		return
	}

	defer state.Deregister()

	config, _, configErr := getConfig(state, url, serverType)

	if configErr != nil {
		// Show the usage of tracebacks and causes
		fmt.Println("Error getting config:", types.GetErrorTraceback(configErr))
		fmt.Println("Error getting config, cause:", types.GetErrorCause(configErr))
		return
	}

	fmt.Println("Obtained config", config)
}

// The main function
// It parses the arguments and executes the correct functions
func main() {
	customURLArg := flag.String("get-custom", "", "The url of a custom server to connect to")
	urlArg := flag.String("get-institute", "", "The url of an institute to connect to")
	secureInternet := flag.String("get-secure", "", "Gets secure internet servers")
	flag.Parse()

	// Connect to a VPN by getting an Institute Access config
	customURLString := *customURLArg
	urlString := *urlArg
	secureInternetString := *secureInternet
	if customURLString != "" {
		printConfig(customURLString, ServerTypeCustom)
		return
	} else if urlString != "" {
		printConfig(urlString, ServerTypeInstituteAccess)
		return
	} else if secureInternetString != "" {
		printConfig(secureInternetString, ServerTypeSecureInternet)
		return
	}
	flag.PrintDefaults()
}
