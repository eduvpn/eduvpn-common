package eduvpn

import (
	"encoding/json"
	"errors"
)

type Server struct {
	BaseURL   string             `json:"base_url"`
	Endpoints *ServerEndpoints   `json:"endpoints"`
	OAuth     *OAuth             `json:"oauth"`
	Profiles  *ServerProfileInfo `json:"profiles"`
}

type ServerProfile struct {
	ID             string   `json:"profile_id"`
	DisplayName    string   `json:"display_name"`
	VPNProtoList   []string `json:"vpn_proto_list"`
	DefaultGateway bool     `json:"default_gateway"`
}

type ServerProfileInfo struct {
	Current uint8 `json:"current_profile"`
	Info    struct {
		ProfileList []ServerProfile `json:"profile_list"`
	} `json:"info"`
}

type ServerEndpointList struct {
	API           string `json:"api_endpoint"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type ServerEndpoints struct {
	API struct {
		V2 ServerEndpointList `json:"http://eduvpn.org/api#2"`
		V3 ServerEndpointList `json:"http://eduvpn.org/api#3"`
	} `json:"api"`
	V string `json:"v"`
}

func (server *Server) Initialize(url string) error {
	server.BaseURL = url
	endpointsErr := server.GetEndpoints()
	if endpointsErr != nil {
		return endpointsErr
	}
	return nil
}

// FIXME: Check validity of tokens
func (server *Server) IsAuthenticated() bool {
	return server.OAuth != nil
}

func (server *Server) GetEndpoints() error {
	url := server.BaseURL + "/.well-known/vpn-user-portal"
	_, body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return bodyErr
	}

	endpoints := &ServerEndpoints{}
	jsonErr := json.Unmarshal(body, &endpoints)

	if jsonErr != nil {
		return jsonErr
	}

	server.Endpoints = endpoints

	return nil
}

func (profiles *ServerProfileInfo) getCurrentProfile() (*ServerProfile, error) {
	if profiles.Info.ProfileList == nil {
		return nil, errors.New("No server profiles")
	}

	if (int)(profiles.Current) >= len(profiles.Info.ProfileList) {
		return nil, errors.New("Invalid profile")
	}
	return &profiles.Info.ProfileList[profiles.Current], nil
}

func (profile *ServerProfile) supportsWireguard() bool {
	for _, proto := range profile.VPNProtoList {
		if proto == "wireguard" {
			return true
		}
	}
	return false
}

func (server *Server) GetCurrentProfile() (*ServerProfile, error) {
	if server.Profiles == nil {
		return nil, errors.New("No server profiles found")
	}

	return server.Profiles.getCurrentProfile()
}

func (server *Server) GetConfig() (string, error) {
	infoErr := server.APIInfo()

	if infoErr != nil {
		return "", infoErr
	}

	profile, profileErr := server.GetCurrentProfile()

	if profileErr != nil {
		return "", profileErr
	}

	if profile.supportsWireguard() {
		return server.WireguardGetConfig()
	}
	return server.OpenVPNGetConfig()
}
