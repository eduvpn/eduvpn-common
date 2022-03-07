package main

import (
	"flag"
	eduvpn "github.com/jwijenbergh/eduvpn-common/src"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func openBrowser(urlString string) {
	log.Printf("OAuth: Initialized with AuthURL %s\n", urlString)
	log.Println("OAuth: Opening browser with xdg-open...")
	exec.Command("xdg-open", urlString).Start()
}

func constructConfig(urlString string) (*oauth2.Config, string) {
	// Get the endpoints
	endpoints, err := eduvpn.APIGetEndpoints(urlString)
	if err != nil {
		log.Fatal("Error API: cannot get endpoints. Message: ", err)
	}
	log.Printf("API: Got endpoints:\n- V3 API %s\n- V3 Authorization %s\n- V3 Token %s\n", endpoints.API.V3.Endpoint, endpoints.API.V3.AuthorizationEndpoint, endpoints.API.V3.TokenEndpoint)

	// Start the OAuth procedure
	config := &oauth2.Config{
		RedirectURL: "http://127.0.0.1:8000/callback",
		ClientID:    "org.eduvpn.app.linux",
		Scopes:      []string{"config"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.API.V3.AuthorizationEndpoint,
			TokenURL: endpoints.API.V3.TokenEndpoint,
		},
	}
	return config, endpoints.API.V3.Endpoint
}

func auth(urlString string) (*http.Client, string) {
	// Get the config
	oauthConfig, apiString := constructConfig(urlString)

	// Initialize oauth with the config
	eduOAuth, err := eduvpn.InitializeOAuth(oauthConfig)
	if err != nil {
		log.Fatal("Error OAuth: cannot initialize OAuth. Message: ", err)
	}

	// Open the browser
	openBrowser(eduOAuth.AuthURL)

	// Get and return authenticated client
	client, err := eduOAuth.GetHTTPTokenClient()
	if err != nil {
		log.Fatal("Error OAUth: cannot get authenticated HTTP client. Message: ", err)
	}
	return client, apiString
}

func show_info(client *http.Client, apiString string) {
	log.Println("OAUth: Got authenticated HTTP client")

	resp, err := client.Get(apiString + "/info")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Println(string(bodyBytes))
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

	client, apiString := auth(urlString)
	show_info(client, apiString)
}
