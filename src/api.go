package eduvpn

import (
	"encoding/json"
	"net/http"
)

type endpointList struct {
	API           string `json:"api_endpoint"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type EduVPNEndpoints struct {
	API struct {
		V2 endpointList `json:"http://eduvpn.org/api#2"`
		V3 endpointList `json:"http://eduvpn.org/api#3"`
	} `json:"api"`
	V string `json:"v"`
}

func APIGetEndpoints(vpnState *EduVPNState) (*EduVPNEndpoints, error) {
	url := vpnState.Server + "/.well-known/vpn-user-portal"
	body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return nil, bodyErr
	}

	structure := &EduVPNEndpoints{}
	jsonErr := json.Unmarshal(body, &structure)

	if jsonErr != nil {
		return nil, jsonErr
	}

	return structure, nil
}

func APIAuthenticatedInfo(vpnState *EduVPNState) (string, error) {
	url := vpnState.Endpoints.API.V3.API + "/info"

	headers := &http.Header{"Authorization": {"Bearer " + vpnState.OAuthToken.Access}}
	body, bodyErr := HTTPGetWithOptionalParams(url, &HTTPOptionalParams{Headers: headers})
	if bodyErr != nil {
		return "", bodyErr
	}
	return string(body), nil
}
