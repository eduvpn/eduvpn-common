package internal

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Server struct {
	BaseURL     string            `json:"base_url"`
	Endpoints   ServerEndpoints   `json:"endpoints"`
	OAuth       OAuth             `json:"oauth"`
	Profiles    ServerProfileInfo `json:"profiles"`
	ProfilesRaw string            `json:"profiles_raw"`
	Logger      *FileLogger       `json:"-"`
	FSM         *FSM              `json:"-"`
}

type Servers struct {
	List       map[string]*Server `json:"list"`
	Current    string             `json:"current"`
	SecureHome string             `json:"secure_home"`
}

func (servers *Servers) GetCurrentServer() (*Server, error) {
	if servers.List == nil {
		return nil, &ServerGetCurrentNoMapError{}
	}
	server, exists := servers.List[servers.Current]

	if !exists || server == nil {
		return nil, &ServerGetCurrentNotFoundError{}
	}
	return server, nil
}

func (server *Server) CancelOAuth() {
	server.OAuth.Cancel()
}

func (server *Server) Init(url string, fsm *FSM, logger *FileLogger) error {
	server.BaseURL = url
	server.FSM = fsm
	server.Logger = logger
	server.OAuth.Init(fsm, logger)
	endpointsErr := server.GetEndpoints()
	if endpointsErr != nil {
		return &ServerInitializeError{URL: url, Err: endpointsErr}
	}
	return nil
}

func (server *Server) EnsureTokens() error {
	if server.OAuth.NeedsRelogin() {
		server.Logger.Log(LOG_INFO, "OAuth: Tokens are invalid, relogging in")
		return server.Login()
	}
	return nil
}

func (servers *Servers) EnsureServer(url string, fsm *FSM, logger *FileLogger, makeCurrent bool) (*Server, error) {
	if url == "" {
		return nil, &ServerEnsureServerEmptyURLError{}
	}
	if servers.List == nil {
		servers.List = make(map[string]*Server)
	}

	server, exists := servers.List[url]

	if !exists || server == nil {
		server = &Server{}
	}
	serverInitErr := server.Init(url, fsm, logger)

	if serverInitErr != nil {
		return nil, &ServerEnsureServerError{Err: serverInitErr}
	}
	servers.List[url] = server

	if makeCurrent {
		servers.Current = url
	}
	return server, nil
}

func (servers *Servers) getSecureInternetHome() (*Server, error) {
	server, exists := servers.List[servers.SecureHome]

	if !exists || server == nil {
		return nil, &ServerGetSecureInternetHomeError{}
	}

	return server, nil
}

func (servers *Servers) EnsureSecureHome(server *Server) {
	if servers.SecureHome == "" {
		servers.SecureHome = server.BaseURL
	}
}

func (servers *Servers) CopySecureInternetOAuth(server *Server) error {
	secureHome, secureHomeErr := servers.getSecureInternetHome()

	if secureHomeErr != nil {
		return &ServerCopySecureInternetOAuthError{Err: secureHomeErr}
	}

	// Forward token properties
	server.OAuth = secureHome.OAuth
	return nil
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

func (server *Server) Login() error {
	return server.OAuth.Login("org.eduvpn.app.linux", server.Endpoints.API.V3.Authorization, server.Endpoints.API.V3.Token)
}

func (server *Server) NeedsRelogin() bool {
	// Check if OAuth needs relogin
	return server.OAuth.NeedsRelogin()
}

func (server *Server) GetEndpoints() error {
	url := server.BaseURL + "/.well-known/vpn-user-portal"
	_, body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return &ServerGetEndpointsError{Err: bodyErr}
	}

	endpoints := ServerEndpoints{}
	jsonErr := json.Unmarshal(body, &endpoints)

	if jsonErr != nil {
		return &ServerGetEndpointsError{Err: jsonErr}
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
	profileID := server.Profiles.Current
	for _, profile := range server.Profiles.Info.ProfileList {
		if profile.ID == profileID {
			return &profile, nil
		}
	}
	return nil, &ServerGetCurrentProfileNotFoundError{ProfileID: profileID}
}

func (server *Server) getConfigWithProfile() (string, error) {
	if !server.FSM.HasTransition(HAS_CONFIG) {
		return "", &FSMWrongStateTransitionError{Got: server.FSM.Current, Want: HAS_CONFIG}
	}
	profile, profileErr := server.getCurrentProfile()

	if profileErr != nil {
		return "", &ServerGetConfigWithProfileError{Err: profileErr}
	}

	if profile.supportsWireguard() {
		return server.WireguardGetConfig()
	}
	return server.OpenVPNGetConfig()
}

func (server *Server) askForProfileID() error {
	if !server.FSM.HasTransition(ASK_PROFILE) {
		return &FSMWrongStateTransitionError{Got: server.FSM.Current, Want: ASK_PROFILE}
	}
	server.FSM.GoTransitionWithData(ASK_PROFILE, server.ProfilesRaw, false)
	return nil
}

func (server *Server) GetConfig() (string, error) {
	if !server.FSM.InState(REQUEST_CONFIG) {
		return "", errors.New(fmt.Sprintf("cannot get a config, invalid state %s", server.FSM.Current.String()))
	}
	infoErr := server.APIInfo()

	if infoErr != nil {
		return "", &ServerGetConfigError{Err: infoErr}
	}

	// Set the current profile if there is only one profile
	if len(server.Profiles.Info.ProfileList) == 1 {
		server.Profiles.Current = server.Profiles.Info.ProfileList[0].ID
		return server.getConfigWithProfile()
	}

	profileErr := server.askForProfileID()

	if profileErr != nil {
		return "", &ServerGetConfigError{Err: profileErr}
	}

	return server.getConfigWithProfile()
}

type ServerGetCurrentProfileNotFoundError struct {
	ProfileID string
}

func (e *ServerGetCurrentProfileNotFoundError) Error() string {
	return fmt.Sprintf("failed to get current profile, profile with ID: %s not found", e.ProfileID)
}

type ServerGetConfigWithProfileError struct {
	Err error
}

func (e *ServerGetConfigWithProfileError) Error() string {
	return fmt.Sprintf("failed to get config including profile with error %v", e.Err)
}

type ServerGetEndpointsError struct {
	Err error
}

func (e *ServerGetEndpointsError) Error() string {
	return fmt.Sprintf("failed to get server endpoint with error %v", e.Err)
}

type ServerGetSecureInternetHomeError struct{}

func (e *ServerGetSecureInternetHomeError) Error() string {
	return "failed to get secure internet home server, not found"
}

type ServerCopySecureInternetOAuthError struct {
	Err error
}

func (e *ServerCopySecureInternetOAuthError) Error() string {
	return fmt.Sprintf("failed to copy oauth tokens from home server with error %v", e.Err)
}

type ServerEnsureServerEmptyURLError struct{}

func (e *ServerEnsureServerEmptyURLError) Error() string {
	return "failed ensuring server, empty url provided"
}

type ServerEnsureServerError struct {
	Err error
}

func (e *ServerEnsureServerError) Error() string {
	return fmt.Sprintf("failed ensuring server with error %v", e.Err)
}

type ServerGetCurrentNoMapError struct{}

func (e *ServerGetCurrentNoMapError) Error() string {
	return "failed getting current server, no servers available"
}

type ServerGetCurrentNotFoundError struct{}

func (e *ServerGetCurrentNotFoundError) Error() string {
	return "failed getting current server, not found"
}

type ServerGetConfigError struct {
	Err error
}

func (e *ServerGetConfigError) Error() string {
	return fmt.Sprintf("failed getting server config with error %v", e.Err)
}

type ServerInitializeError struct {
	URL string
	Err error
}

func (e *ServerInitializeError) Error() string {
	return fmt.Sprintf("failed initializing server with url %s and error %v", e.URL, e.Err)
}
