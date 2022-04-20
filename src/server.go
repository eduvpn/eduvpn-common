package eduvpn

import (
	"encoding/json"
	"errors"
)

type Server struct {
	BaseURL     string            `json:"base_url"`
	Endpoints   ServerEndpoints   `json:"endpoints"`
	OAuth       OAuth             `json:"oauth"`
	Profiles    ServerProfileInfo `json:"profiles"`
	ProfilesRaw string            `json:"profiles_raw"`
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
	// Check if OAuth needs relogin
	return server.OAuth.NeedsRelogin()
}

func (server *Server) GetEndpoints() error {
	url := server.BaseURL + "/.well-known/vpn-user-portal"
	_, body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return bodyErr
	}

	endpoints := ServerEndpoints{}
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

func (server *Server) getCurrentProfile() (*ServerProfile, error) {
	profile_id := server.Profiles.Current
	for _, profile := range server.Profiles.Info.ProfileList {
		if profile.ID == profile_id {
			return &profile, nil
		}
	}
	return nil, errors.New("no profile found for id")
}

func (server *Server) getConfigWithProfile() (string, error) {
	if !GetVPNState().HasTransition(HAS_CONFIG) {
		return "", errors.New("cannot get a config with a profile, invalid state")
	}
	profile, profileErr := server.getCurrentProfile()

	if profileErr != nil {
		return "", profileErr
	}

	if profile.supportsWireguard() {
		return server.WireguardGetConfig()
	}
	return server.OpenVPNGetConfig()
}

func (server *Server) askForProfileID() error {
	if !GetVPNState().HasTransition(ASK_PROFILE) {
		return errors.New("cannot ask for a profile id, invalid state")
	}
	GetVPNState().GoTransition(ASK_PROFILE, server.ProfilesRaw)
	return nil
}

func (server *Server) GetConfig() (string, error) {
	if !GetVPNState().InState(REQUEST_CONFIG) {
		return "", errors.New("cannot get a config, invalid state")
	}
	infoErr := server.APIInfo()

	if infoErr != nil {
		return "", infoErr
	}

	// Set the current profile if there is only one profile
	if len(server.Profiles.Info.ProfileList) == 1 {
		server.Profiles.Current = server.Profiles.Info.ProfileList[0].ID
		return server.getConfigWithProfile()
	}

	profileErr := server.askForProfileID()

	if profileErr != nil {
		return "", nil
	}

	return server.getConfigWithProfile()
}
