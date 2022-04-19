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
	ProfilesRaw string           `json:"profiles_raw"`
}

type ServerProfile struct {
	ID             string   `json:"profile_id"`
	DisplayName    string   `json:"display_name"`
	VPNProtoList   []string `json:"vpn_proto_list"`
	DefaultGateway bool     `json:"default_gateway"`
}

type ServerProfileInfo struct {
	Current string `json:"current_profile"`
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
	if !GetVPNState().HasTransition(CHOSEN_SERVER) {
		return errors.New("cannot choose a server")
	}
	server.BaseURL = url
	endpointsErr := server.GetEndpoints()
	if endpointsErr != nil {
		return endpointsErr
	}
	GetVPNState().GoTransition(CHOSEN_SERVER, "Chosen server")
	return nil
}

func (server *Server) NeedsRelogin() bool {
	// Server has no oauth tokens
	if server.OAuth == nil || server.OAuth.Token == nil {
		return true
	}

	// Server has oauth tokens, check if they need a relogin
	return server.OAuth.NeedsRelogin()
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

func (profile *ServerProfile) supportsWireguard() bool {
	for _, proto := range profile.VPNProtoList {
		if proto == "wireguard" {
			return true
		}
	}
	return false
}

func (server *Server) getProfileForID(profile_id string) (*ServerProfile, error) {
	for _, profile := range server.Profiles.Info.ProfileList {
		if profile.ID == profile_id {
			return &profile, nil
		}
	}
	return nil, errors.New("no profile found for id")
}

func (server *Server) getConfigWithProfile(profile_id string) (string, error) {
	profile, profileErr := server.getProfileForID(profile_id)

	if profileErr != nil {
		return "", profileErr
	}

	if profile.supportsWireguard() {
		return server.WireguardGetConfig(profile_id)
	}
	return server.OpenVPNGetConfig(profile_id)
}

func (server *Server) askForProfileID() (string, error) {
	_, profile_id := GetVPNState().GoTransition(ASK_PROFILE, server.ProfilesRaw)
	return profile_id, nil
}

func (server *Server) GetConfig() (string, error) {
	infoErr := server.APIInfo()

	if infoErr != nil {
		return "", infoErr
	}

	// Set the current profile if there is only one profile
	if len(server.Profiles.Info.ProfileList) == 1 {
		return server.getConfigWithProfile(server.Profiles.Info.ProfileList[0].ID)
	}

	profile_id, profileErr := server.askForProfileID()

	if profileErr != nil {
		return "", nil
	}

	return server.getConfigWithProfile(profile_id)
}
