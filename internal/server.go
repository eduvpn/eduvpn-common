package internal

import (
	"encoding/json"
	"fmt"
)

// The base type for servers
type ServerBase struct {
	URL         string            `json:"base_url"`
	Endpoints   ServerEndpoints   `json:"endpoints"`
	Profiles    ServerProfileInfo `json:"profiles"`
	ProfilesRaw string            `json:"profiles_raw"`
	Logger      *FileLogger       `json:"-"`
	FSM         *FSM              `json:"-"`
	StartTime   int64             `json:"start-time"`
	EndTime     int64             `json:"end-time"`
}

// An instute access server
type InstituteAccessServer struct {
	// An instute access server has its own OAuth
	OAuth OAuth `json:"oauth"`

	// Embed the server base
	Base ServerBase `json:"base"`
}

// A secure internet server which has its own OAuth tokens
// It specifies the current location url it is connected to
type SecureInternetHomeServer struct {
	OAuth OAuth `json:"oauth"`

	// The home server has a list of info for each configured server
	BaseMap map[string]*ServerBase `json:"base_map"`

	// We have the home url and the current url
	HomeURL    string `json:"home_url"`
	CurrentURL string `json:"current_url"`
}

type InstituteServers struct {
	Map        map[string]*InstituteAccessServer `json:"map"`
	CurrentURL string                            `json:"current_url"`
}

func (servers *Servers) GetCurrentServer() (Server, error) {
	if servers.IsSecureInternet {
		return &servers.SecureInternetHomeServer, nil
	}
	currentInstitute := servers.InstituteServers.CurrentURL
	institutes := servers.InstituteServers.Map
	if institutes == nil {
		return nil, &ServerGetCurrentNoMapError{}
	}
	institute, exists := institutes[currentInstitute]

	if !exists || institute == nil {
		return nil, &ServerGetCurrentNotFoundError{}
	}
	return institute, nil
}

type Servers struct {
	InstituteServers         InstituteServers         `json:"institute_servers"`
	SecureInternetHomeServer SecureInternetHomeServer `json:"secure_internet_home"`
	IsSecureInternet         bool                     `json:"is_secure_internet"`
}

type Server interface {
	// Gets the current OAuth object
	GetOAuth() *OAuth

	// Gets the server base
	GetBase() (*ServerBase, error)

	// initialize method
	init(url string, fsm *FSM, logger *FileLogger) error
}

// For an institute, we can simply get the OAuth
func (institute *InstituteAccessServer) GetOAuth() *OAuth {
	return &institute.OAuth
}

func (secure *SecureInternetHomeServer) GetOAuth() *OAuth {
	return &secure.OAuth
}

func (institute *InstituteAccessServer) GetBase() (*ServerBase, error) {
	return &institute.Base, nil
}

func (server *SecureInternetHomeServer) GetBase() (*ServerBase, error) {
	if server.BaseMap == nil {
		return nil, &ServerSecureInternetMapNotFoundError{}
	}

	base, exists := server.BaseMap[server.CurrentURL]

	if !exists {
		return nil, &ServerSecureInternetBaseNotFoundError{Current: server.CurrentURL}
	}
	return base, nil
}

func (institute *InstituteAccessServer) init(url string, fsm *FSM, logger *FileLogger) error {
	institute.Base.URL = url
	institute.Base.FSM = fsm
	institute.Base.Logger = logger
	endpoints, endpointsErr := getEndpoints(url)
	if endpointsErr != nil {
		return &ServerInitializeError{URL: url, Err: endpointsErr}
	}
	institute.OAuth.Init(endpoints.API.V3.Authorization, endpoints.API.V3.Token, fsm, logger)
	institute.Base.Endpoints = *endpoints
	return nil
}

func (secure *SecureInternetHomeServer) init(url string, fsm *FSM, logger *FileLogger) error {
	// Initialize the base map if it is non-nil
	if secure.BaseMap == nil {
		secure.BaseMap = make(map[string]*ServerBase)
	}

	// Add it if not present
	base, exists := secure.BaseMap[url]

	if !exists || base == nil {
		// Create the base to be added to the map
		base = &ServerBase{}
		base.URL = url
		endpoints, endpointsErr := getEndpoints(url)
		if endpointsErr != nil {
			return &ServerInitializeError{URL: url, Err: endpointsErr}
		}
		base.Endpoints = *endpoints
	}

	// Pass the fsm and logger
	base.FSM = fsm
	base.Logger = logger

	// Ensure it is in the map
	secure.BaseMap[url] = base

	// Set the home url if it is not set yet
	if secure.HomeURL == "" {
		secure.HomeURL = url
		// Make sure oauth contains our endpoints
		secure.OAuth.Init(base.Endpoints.API.V3.Authorization, base.Endpoints.API.V3.Token, fsm, logger)
	} else { // Else just pass in the fsm and logger
		secure.OAuth.Update(fsm, logger)
	}

	// Set the current url
	secure.CurrentURL = url
	return nil
}

func Login(server Server) error {
	return server.GetOAuth().Login("org.eduvpn.app.linux")
}

func EnsureTokens(server Server) error {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &ServerEnsureTokensError{Err: baseErr}
	}
	if server.GetOAuth().NeedsRelogin() {
		base.Logger.Log(LOG_INFO, "OAuth: Tokens are invalid, relogging in")
		loginErr := Login(server)

		if loginErr != nil {
			return &ServerEnsureTokensError{Err: loginErr}
		}
	}
	return nil
}

func NeedsRelogin(server Server) bool {
	return server.GetOAuth().NeedsRelogin()
}

func CancelOAuth(server Server) {
	server.GetOAuth().Cancel()
}

func (servers *Servers) EnsureServer(url string, isSecureInternet bool, fsm *FSM, logger *FileLogger) (Server, error) {
	// Intialize the secure internet server
	// This calls the init method which takes care of the rest
	if isSecureInternet {
		initErr := servers.SecureInternetHomeServer.init(url, fsm, logger)

		if initErr != nil {
			return nil, &ServerEnsureServerError{Err: initErr}
		}

		servers.IsSecureInternet = true
		return &servers.SecureInternetHomeServer, nil
	}

	instituteServers := &servers.InstituteServers

	if instituteServers.Map == nil {
		instituteServers.Map = make(map[string]*InstituteAccessServer)
	}

	institute, exists := instituteServers.Map[url]

	// initialize the server if it doesn't exist yet
	if !exists {
		institute = &InstituteAccessServer{}
	}

	// Set the current server
	instituteServers.CurrentURL = url
	instituteInitErr := institute.init(url, fsm, logger)
	if instituteInitErr != nil {
		return nil, &ServerEnsureServerError{Err: instituteInitErr}
	}
	instituteServers.Map[url] = institute
	servers.IsSecureInternet = false
	return institute, nil
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

// Make this a var which we can overwrite in the tests
var WellKnownPath string = ".well-known/vpn-user-portal"

func getEndpoints(baseURL string) (*ServerEndpoints, error) {
	url := fmt.Sprintf("%s/%s", baseURL, WellKnownPath)
	_, body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return nil, &ServerGetEndpointsError{Err: bodyErr}
	}

	endpoints := &ServerEndpoints{}
	jsonErr := json.Unmarshal(body, endpoints)

	if jsonErr != nil {
		return nil, &ServerGetEndpointsError{Err: jsonErr}
	}

	return endpoints, nil
}

func (profile *ServerProfile) supportsProtocol(protocol string) bool {
	for _, proto := range profile.VPNProtoList {
		if proto == protocol {
			return true
		}
	}
	return false
}

func (profile *ServerProfile) supportsWireguard() bool {
	return profile.supportsProtocol("wireguard")
}

func (profile *ServerProfile) supportsOpenVPN() bool {
	return profile.supportsProtocol("openvpn")
}

func getCurrentProfile(server Server) (*ServerProfile, error) {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return nil, &ServerGetCurrentProfileError{Err: baseErr}
	}
	profileID := base.Profiles.Current
	for _, profile := range base.Profiles.Info.ProfileList {
		if profile.ID == profileID {
			return &profile, nil
		}
	}
	return nil, &ServerGetCurrentProfileNotFoundError{ProfileID: profileID}
}

func getConfigWithProfile(server Server, forceTCP bool) (string, string, error) {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &ServerGetConfigWithProfileError{Err: baseErr}
	}
	if !base.FSM.HasTransition(HAS_CONFIG) {
		return "", "", &FSMWrongStateTransitionError{Got: base.FSM.Current, Want: HAS_CONFIG}
	}
	profile, profileErr := getCurrentProfile(server)

	if profileErr != nil {
		return "", "", &ServerGetConfigWithProfileError{Err: profileErr}
	}

	supportsOpenVPN := profile.supportsOpenVPN()
	supportsWireguard := profile.supportsWireguard()

	// If forceTCP we must be able to get a config with OpenVPN
	if forceTCP && supportsOpenVPN {
		return "", "", &ServerGetConfigForceTCPError{}
	}

	var config string
	var configType string
	var configErr error

	if supportsWireguard {
		// A wireguard connect call needs to generate a wireguard key and add it to the config
		// Also the server could send back an OpenVPN config if it supports OpenVPN
		config, configType, configErr = WireguardGetConfig(server, supportsOpenVPN)
	} else {
		config, configType, configErr = OpenVPNGetConfig(server)
	}

	if configErr != nil {
		return "", "", &ServerGetConfigWithProfileError{Err: configErr}
	}

	return config, configType, nil
}

func askForProfileID(server Server) error {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &ServerAskForProfileIDError{Err: baseErr}
	}
	if !base.FSM.HasTransition(ASK_PROFILE) {
		return &FSMWrongStateTransitionError{Got: base.FSM.Current, Want: ASK_PROFILE}
	}
	base.FSM.GoTransitionWithData(ASK_PROFILE, base.ProfilesRaw, false)
	return nil
}

func GetConfig(server Server, forceTCP bool) (string, string, error) {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &ServerGetConfigError{Err: baseErr}
	}
	if !base.FSM.InState(REQUEST_CONFIG) {
		return "", "", &FSMWrongStateError{Got: base.FSM.Current, Want: REQUEST_CONFIG}
	}

	// Get new profiles using the info call
	// This does not override the current profile
	infoErr := APIInfo(server)
	if infoErr != nil {
		return "", "", &ServerGetConfigError{Err: infoErr}
	}

	// If there was a profile chosen and it doesn't exist anymore, reset it
	if base.Profiles.Current != "" {
		_, existsProfileErr := getCurrentProfile(server)
		if existsProfileErr != nil {
			base.Logger.Log(LOG_INFO, fmt.Sprintf("Profile %s no longer exists, resetting the profile", base.Profiles.Current))
			base.Profiles.Current = ""
		}
	}

	// Set the current profile if there is only one profile or profile is already selected
	if len(base.Profiles.Info.ProfileList) == 1 || base.Profiles.Current != "" {
		// Set the first profile if none is selected
		if base.Profiles.Current == "" {
			base.Profiles.Current = base.Profiles.Info.ProfileList[0].ID
		}
		return getConfigWithProfile(server, forceTCP)
	}

	profileErr := askForProfileID(server)

	if profileErr != nil {
		return "", "", &ServerGetConfigError{Err: profileErr}
	}

	return getConfigWithProfile(server, forceTCP)
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

type ServerGetConfigForceTCPError struct{}

func (e *ServerGetConfigForceTCPError) Error() string {
	return fmt.Sprintf("failed to get config, force TCP is on but the server does not support OpenVPN")
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

type ServerInstituteBaseNotFoundError struct {
	Err error
}

func (e *ServerInstituteBaseNotFoundError) Error() string {
	return "institute base not found"
}

type ServerSecureInternetMapNotFoundError struct{}

func (e *ServerSecureInternetMapNotFoundError) Error() string {
	return "secure internet map not found"
}

type ServerSecureInternetBaseNotFoundError struct {
	Current string
}

func (e *ServerSecureInternetBaseNotFoundError) Error() string {
	return fmt.Sprintf("secure internet base not found with current: %s", e.Current)
}

type ServerGetCurrentProfileError struct {
	Err error
}

func (e *ServerGetCurrentProfileError) Error() string {
	return fmt.Sprintf("failed getting current profile with error: %v", e.Err)
}

type ServerAskForProfileIDError struct {
	Err error
}

func (e *ServerAskForProfileIDError) Error() string {
	return fmt.Sprintf("ask for profile ID error: %v", e.Err)
}

type ServerEnsureTokensError struct {
	Err error
}

func (e *ServerEnsureTokensError) Error() string {
	return fmt.Sprintf("failed ensuring tokens with error: %v", e.Err)
}
