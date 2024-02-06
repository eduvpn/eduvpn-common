package endpoints

import (
	"fmt"
	"net/url"
)

type List struct {
	API           string `json:"api_endpoint"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
}

type Versions struct {
	V2 List `json:"http://eduvpn.org/api#2"`
	V3 List `json:"http://eduvpn.org/api#3"`
}

// Endpoints defines the json format for /.well-known/vpn-user-portal".
type Endpoints struct {
	API Versions `json:"api"`
	V   string   `json:"v"`
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
