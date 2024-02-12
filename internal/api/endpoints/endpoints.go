// Package endpoints defines a wrapper around the various
// endpoints returned by an eduVPN server in well-known
package endpoints

import (
	"fmt"
	"net/url"
)

// List is the list of endpoints as returned by the eduVPN server
type List struct {
	// API is the API endpoint which we use for calls such as /info, /connect, ...
	API string `json:"api_endpoint"`
	// Authorization is the authorization endpoint for OAuth
	Authorization string `json:"authorization_endpoint"`
	// Token is the token endpoint for OAuth
	Token string `json:"token_endpoint"`
}

// Versions is the endpoints separated by API version
type Versions struct {
	// V2 is the legacy V2 API, this is not used
	V2 List `json:"http://eduvpn.org/api#2"`
	// V3 is the newest API, which we use
	V3 List `json:"http://eduvpn.org/api#3"`
}

// Endpoints defines the json format for /.well-known/vpn-user-portal".
type Endpoints struct {
	// API defines the API endpoints, split by version
	API Versions `json:"api"`
	// V is the version string for the server
	V string `json:"v"`
}

// Validate validates the endpoints by parsing them and checking the scheme is HTTP
// An error is returned if they are not valid
func (e Endpoints) Validate() error {
	v3 := e.API.V3
	pAPI, err := url.Parse(v3.API)
	if err != nil {
		return fmt.Errorf("failed to parse API endpoint: %w", err)
	}
	pAuth, err := url.Parse(v3.Authorization)
	if err != nil {
		return fmt.Errorf("failed to parse API authorization endpoint: %w", err)
	}
	pToken, err := url.Parse(v3.Token)
	if err != nil {
		return fmt.Errorf("failed to parse API token endpoint: %w", err)
	}
	if pAPI.Scheme != "https" {
		return fmt.Errorf("API Scheme: '%s', is not equal to HTTPS", pAPI.Scheme)
	}
	if pAPI.Scheme != pAuth.Scheme {
		return fmt.Errorf("API scheme: '%v', is not equal to authorization scheme: '%v'", pAPI.Scheme, pAuth.Scheme)
	}
	if pAPI.Scheme != pToken.Scheme {
		return fmt.Errorf("API scheme: '%v', is not equal to token scheme: '%v'", pAPI.Scheme, pToken.Scheme)
	}
	return nil
}
