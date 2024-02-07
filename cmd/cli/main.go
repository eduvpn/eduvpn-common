package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/eduvpn/eduvpn-common/client"
	"github.com/eduvpn/eduvpn-common/internal/version"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	srvtypes "github.com/eduvpn/eduvpn-common/types/server"

	"github.com/pkg/browser"
)

// Open a browser with xdg-open.
func openBrowser(data interface{}) {
	str, ok := data.(string)
	if !ok {
		return
	}
	fmt.Printf("OAuth: Authorization URL: %s\n", str)
	fmt.Println("Opening browser...")
	go func() {
		err := browser.OpenURL(str)
		if err != nil {
			fmt.Fprintln(os.Stderr, "failed to open browser with error:", err)
			fmt.Println("Please open your browser manually")
		}
	}()
}

// GetLanguageMatched uses a map from language tags to strings to extract the right language given the tag
// It implements it according to https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY.md#language-matching
func GetLanguageMatched(langMap map[string]string, langTag string) string {
	// If no map is given, return the empty string
	if len(langMap) == 0 {
		return ""
	}
	// Try to find the exact match
	if val, ok := langMap[langTag]; ok {
		return val
	}
	// Try to find a key that starts with the OS language setting
	for k := range langMap {
		if strings.HasPrefix(k, langTag) {
			return langMap[k]
		}
	}
	// Try to find a key that starts with the first part of the OS language (e.g. de-)
	pts := strings.Split(langTag, "-")
	// We have a "-"
	if len(pts) > 1 {
		for k := range langMap {
			if strings.HasPrefix(k, pts[0]+"-") {
				return langMap[k]
			}
		}
	}
	// search for just the language (e.g. de)
	for k := range langMap {
		if k == pts[0] {
			return langMap[k]
		}
	}

	// Pick one that is deemed best, e.g. en-US or en, but note that not all languages are always available!
	// We force an entry that is english exactly or with an english prefix
	for k := range langMap {
		if k == "en" || strings.HasPrefix(k, "en-") {
			return langMap[k]
		}
	}

	// Otherwise just return one
	for k := range langMap {
		return langMap[k]
	}

	return ""
}

// Ask for a profile in the command line.
func sendProfile(state *client.Client, data interface{}) {
	fmt.Printf("Multiple VPN profiles found. Please select a profile by entering e.g. 1")
	d, ok := data.(*srvtypes.RequiredAskTransition)
	if !ok {
		fmt.Fprintf(os.Stderr, "\ninvalid data type: %v\n", reflect.TypeOf(data))
		return
	}
	sps, ok := d.Data.(srvtypes.Profiles)
	if !ok {
		fmt.Fprintf(os.Stderr, "\ninvalid data type for profiles: %v\n", reflect.TypeOf(d.Data))
		return
	}

	ps := ""
	var options []string
	i := 0
	for k, v := range sps.Map {
		ps += fmt.Sprintf("\n%d - %s", i+1, GetLanguageMatched(v.DisplayName, "en"))
		options = append(options, k)
		i++
	}

	// Show the profiles
	fmt.Println(ps)

	var idx int
	if _, err := fmt.Scanf("%d", &idx); err != nil || idx <= 0 ||
		idx > len(sps.Map) {
		fmt.Fprintln(os.Stderr, "invalid profile chosen, please retry")
		sendProfile(state, data)
		return
	}

	p := options[idx-1]
	fmt.Println("Sending profile ID", p)
	if err := d.C.Send(p); err != nil {
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
func getConfig(state *client.Client, url string, srvType srvtypes.Type) (*srvtypes.Configuration, error) {
	if !strings.HasPrefix(url, "http") {
		url = "https://" + url
	}
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	err := state.AddServer(ck, url, srvType, false)
	if err != nil {
		return nil, err
	}
	return state.GetConfig(ck, url, srvType, false, false)
}

// Get a config for a single server, Institute Access or Secure Internet.
func printConfig(url string, srvType srvtypes.Type) {
	var c *client.Client
	c, err := client.New(
		"org.eduvpn.app.linux",
		fmt.Sprintf("%s-cli", version.Version),
		"configs",
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
	_ = c.Register()

	ck := cookie.NewWithContext(context.Background())
	_, err = c.DiscoOrganizations(ck)
	if err != nil {
		panic(err)
	}
	_, err = c.DiscoServers(ck)
	if err != nil {
		panic(err)
	}

	defer c.Deregister()

	cfg, err := getConfig(c, url, srvType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed getting a config: %v\n", err)
		return
	}
	fmt.Println(cfg.Protocol)
	fmt.Println("Obtained config:", cfg.VPNConfig)
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
		printConfig(*cu, srvtypes.TypeCustom)
	case *u != "":
		printConfig(*u, srvtypes.TypeInstituteAccess)
	case *sec != "":
		printConfig(*sec, srvtypes.TypeSecureInternet)
	default:
		flag.PrintDefaults()
	}
}
