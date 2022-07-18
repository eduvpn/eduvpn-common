package server

import (
	"encoding/json"
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/oauth"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
	"github.com/jwijenbergh/eduvpn-common/internal/wireguard"
)

// The base type for servers
type ServerBase struct {
	URL            string            `json:"base_url"`
	DisplayName    map[string]string `json:"display_name"`
	SupportContact []string          `json:"support_contact"`
	Endpoints      ServerEndpoints   `json:"endpoints"`
	Profiles       ServerProfileInfo `json:"profiles"`
	ProfilesRaw    string            `json:"profiles_raw"`
	StartTime      int64             `json:"start_time"`
	EndTime        int64             `json:"expire_time"`
	Type           string            `json:"server_type"`
	Logger         *log.FileLogger   `json:"-"`
	FSM            *fsm.FSM          `json:"-"`
}

// An instute access server
type InstituteAccessServer struct {
	// An instute access server has its own OAuth
	OAuth oauth.OAuth `json:"oauth"`

	// Embed the server base
	Base ServerBase `json:"base"`
}

// A secure internet server which has its own OAuth tokens
// It specifies the current location url it is connected to
type SecureInternetHomeServer struct {
	DisplayName map[string]string `json:"display_name"`
	OAuth       oauth.OAuth       `json:"oauth"`

	// The home server has a list of info for each configured server location
	BaseMap map[string]*ServerBase `json:"base_map"`

	// We have the authorization URL template, the home organization ID and the current location
	AuthorizationTemplate string `json:"authorization_template"`
	HomeOrganizationID    string `json:"home_organization_id"`
	CurrentLocation       string `json:"current_location"`
}

type InstituteServers struct {
	Map        map[string]*InstituteAccessServer `json:"map"`
	CurrentURL string                            `json:"current_url"`
}

func (servers *Servers) GetCurrentServer() (Server, error) {
	errorMessage := "failed getting current server"
	if servers.IsSecureInternet {
		if !servers.HasSecureLocation() {
			return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerGetCurrentNotFoundError{}}
		}
		return &servers.SecureInternetHomeServer, nil
	}
	currentInstitute := servers.InstituteServers.CurrentURL
	institutes := servers.InstituteServers.Map
	if institutes == nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerGetCurrentNoMapError{}}
	}
	institute, exists := institutes[currentInstitute]

	if !exists || institute == nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerGetCurrentNotFoundError{}}
	}
	return institute, nil
}

func (servers *Servers) GetJSON() (string, error) {
	bytes, bytesErr := json.Marshal(servers)

	if bytesErr != nil {
		return "", bytesErr
	}

	return string(bytes), nil
}

type Servers struct {
	InstituteServers         InstituteServers         `json:"institute_servers"`
	SecureInternetHomeServer SecureInternetHomeServer `json:"secure_internet_home"`
	IsSecureInternet         bool                     `json:"is_secure_internet"`
}

type Server interface {
	// Gets the current OAuth object
	GetOAuth() *oauth.OAuth

	// Get the authorization URL template function
	GetTemplateAuth() func(string) string

	// Gets the server base
	GetBase() (*ServerBase, error)
}

// For an institute, we can simply get the OAuth
func (institute *InstituteAccessServer) GetOAuth() *oauth.OAuth {
	return &institute.OAuth
}

func (secure *SecureInternetHomeServer) GetOAuth() *oauth.OAuth {
	return &secure.OAuth
}

func (institute *InstituteAccessServer) GetTemplateAuth() func(string) string {
	return func(authURL string) string {
		return authURL
	}
}

func (secure *SecureInternetHomeServer) GetTemplateAuth() func(string) string {
	return func(authURL string) string {
		return util.ReplaceWAYF(secure.AuthorizationTemplate, authURL, secure.HomeOrganizationID)
	}
}

func (institute *InstituteAccessServer) GetBase() (*ServerBase, error) {
	return &institute.Base, nil
}

func (server *SecureInternetHomeServer) GetBase() (*ServerBase, error) {
	errorMessage := "failed getting current secure internet home base"
	if server.BaseMap == nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerSecureInternetMapNotFoundError{}}
	}

	base, exists := server.BaseMap[server.CurrentLocation]

	if !exists {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerSecureInternetBaseNotFoundError{Current: server.CurrentLocation}}
	}
	return base, nil
}

func (institute *InstituteAccessServer) init(url string, displayName map[string]string, serverType string, supportContact []string, fsm *fsm.FSM, logger *log.FileLogger) error {
	errorMessage := fmt.Sprintf("failed initializing institute server %s", url)
	institute.Base.URL = url
	institute.Base.DisplayName = displayName
	institute.Base.SupportContact = supportContact
	institute.Base.FSM = fsm
	institute.Base.Logger = logger
	institute.Base.Type = serverType
	endpoints, endpointsErr := APIGetEndpoints(url)
	if endpointsErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: endpointsErr}
	}
	institute.OAuth.Init(endpoints.API.V3.Authorization, endpoints.API.V3.Token, fsm, logger)
	institute.Base.Endpoints = *endpoints
	return nil
}

func (servers *Servers) HasSecureLocation() bool {
	return servers.SecureInternetHomeServer.CurrentLocation != ""
}

func (secure *SecureInternetHomeServer) addLocation(locationServer *types.DiscoveryServer, fsm *fsm.FSM, logger *log.FileLogger) (*ServerBase, error) {
	errorMessage := "failed adding a location"
	// Initialize the base map if it is non-nil
	if secure.BaseMap == nil {
		secure.BaseMap = make(map[string]*ServerBase)
	}

	// Add the location to the base map
	base, exists := secure.BaseMap[locationServer.CountryCode]

	if !exists || base == nil {
		// Create the base to be added to the map
		base = &ServerBase{}
		base.URL = locationServer.BaseURL
		base.DisplayName = secure.DisplayName
		base.SupportContact = locationServer.SupportContact
		base.Type = "secure_internet"
		endpoints, endpointsErr := APIGetEndpoints(locationServer.BaseURL)
		if endpointsErr != nil {
			return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: endpointsErr}
		}
		base.Endpoints = *endpoints
	}

	// Pass the fsm and logger
	base.FSM = fsm
	base.Logger = logger

	// Ensure it is in the map
	secure.BaseMap[locationServer.CountryCode] = base
	return base, nil
}

// Initializes the home server and adds its own location
func (secure *SecureInternetHomeServer) init(homeOrg *types.DiscoveryOrganization, homeLocation *types.DiscoveryServer, fsm *fsm.FSM, logger *log.FileLogger) error {
	errorMessage := "failed initializing secure internet home server"

	if secure.HomeOrganizationID != homeOrg.OrgId {
		// New home organisation, clear everything
		*secure = SecureInternetHomeServer{}
	}

	// Make sure to set the organization ID
	secure.HomeOrganizationID = homeOrg.OrgId
	secure.DisplayName = homeOrg.DisplayName

	// Make sure to set the authorization URL template
	secure.AuthorizationTemplate = homeLocation.AuthenticationURLTemplate

	base, baseErr := secure.addLocation(homeLocation, fsm, logger)

	if baseErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	// Make sure oauth contains our endpoints
	secure.OAuth.Init(base.Endpoints.API.V3.Authorization, base.Endpoints.API.V3.Token, fsm, logger)
	return nil
}

func ShouldRenewButton(server Server) bool {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		// FIXME: Log error here?
		return false
	}

	// Get current time
	current := util.GenerateTimeSeconds()

	// 30 minutes have not passed
	if current <= (base.StartTime + 30*60) {
		return false
	}

	// Session will not expire today
	if current <= (base.EndTime - 24*60*60) {
		return false
	}

	// Session duration is less than 24 hours but not 75% has passed
	duration := base.EndTime - base.StartTime

	// TODO: Is converting to float64 okay here?
	if duration < 24*60*60 && float64(current) <= (float64(base.StartTime)+0.75*float64(duration)) {
		return false
	}

	return true
}

func Login(server Server) error {
	return server.GetOAuth().Login("org.eduvpn.app.linux", server.GetTemplateAuth())
}

func EnsureTokens(server Server) error {
	errorMessage := "failed ensuring server tokens"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	if server.GetOAuth().NeedsRelogin() {
		base.Logger.Log(log.LOG_INFO, "OAuth: Tokens are invalid, relogging in")
		loginErr := Login(server)

		if loginErr != nil {
			return &types.WrappedErrorMessage{Message: errorMessage, Err: loginErr}
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

func (servers *Servers) AddInstituteAccess(instituteServer *types.DiscoveryServer, fsm *fsm.FSM, logger *log.FileLogger) (Server, error) {
	url := instituteServer.BaseURL
	errorMessage := fmt.Sprintf("failed adding institute access server: %s", url)
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
	instituteInitErr := institute.init(url, instituteServer.DisplayName, instituteServer.Type, instituteServer.SupportContact, fsm, logger)
	if instituteInitErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: instituteInitErr}
	}
	instituteServers.Map[url] = institute
	servers.IsSecureInternet = false
	return institute, nil
}

func (servers *Servers) GetSecureLocation() string {
	return servers.SecureInternetHomeServer.CurrentLocation
}

func (servers *Servers) SetSecureLocation(chosenLocationServer *types.DiscoveryServer, fsm *fsm.FSM, logger *log.FileLogger) error {
	errorMessage := "failed to set secure location"
	// Make sure to add the current location
	_, addLocationErr := servers.SecureInternetHomeServer.addLocation(chosenLocationServer, fsm, logger)

	if addLocationErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: addLocationErr}
	}

	servers.SecureInternetHomeServer.CurrentLocation = chosenLocationServer.CountryCode
	return nil
}

func (servers *Servers) AddSecureInternet(secureOrg *types.DiscoveryOrganization, secureServer *types.DiscoveryServer, fsm *fsm.FSM, logger *log.FileLogger) (Server, error) {
	errorMessage := "failed adding secure internet server"
	// If we have specified an organization ID
	// We also need to get an authorization template
	initErr := servers.SecureInternetHomeServer.init(secureOrg, secureServer, fsm, logger)

	if initErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: initErr}
	}

	servers.IsSecureInternet = true
	return &servers.SecureInternetHomeServer, nil
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
	errorMessage := "failed getting current profile"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	profileID := base.Profiles.Current
	for _, profile := range base.Profiles.Info.ProfileList {
		if profile.ID == profileID {
			return &profile, nil
		}
	}

	return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerGetCurrentProfileNotFoundError{ProfileID: profileID}}
}

func wireguardGetConfig(server Server, supportsOpenVPN bool) (string, string, error) {
	errorMessage := "failed getting server WireGuard configuration"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	profile_id := base.Profiles.Current
	wireguardKey, wireguardErr := wireguard.GenerateKey()

	if wireguardErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: wireguardErr}
	}

	wireguardPublicKey := wireguardKey.PublicKey().String()
	config, content, expires, configErr := APIConnectWireguard(server, profile_id, wireguardPublicKey, supportsOpenVPN)

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}

	// Store start and end time
	base.StartTime = util.GenerateTimeSeconds()
	base.EndTime = expires

	if content == "wireguard" {
		// This needs the go code a way to identify a connection
		// Use the uuid of the connection e.g. on Linux
		// This needs the client code to call the go code

		config = wireguard.ConfigAddKey(config, wireguardKey)
	}

	return config, content, nil
}

func openVPNGetConfig(server Server) (string, string, error) {
	errorMessage := "failed getting server OpenVPN configuration"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	profile_id := base.Profiles.Current
	configOpenVPN, expires, configErr := APIConnectOpenVPN(server, profile_id)

	// Store start and end time
	base.StartTime = util.GenerateTimeSeconds()
	base.EndTime = expires

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}

	return configOpenVPN, "openvpn", nil
}

func getConfigWithProfile(server Server, forceTCP bool) (string, string, error) {
	errorMessage := "failed getting an OpenVPN/WireGuard configuration with a profile"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	if !base.FSM.HasTransition(fsm.HAS_CONFIG) {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: fsm.WrongStateTransitionError{Got: base.FSM.Current, Want: fsm.HAS_CONFIG}.CustomError()}
	}
	profile, profileErr := getCurrentProfile(server)

	if profileErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: profileErr}
	}

	supportsOpenVPN := profile.supportsOpenVPN()
	supportsWireguard := profile.supportsWireguard()

	// If forceTCP we must be able to get a config with OpenVPN
	if forceTCP && supportsOpenVPN {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: &ServerGetConfigForceTCPError{}}
	}

	var config string
	var configType string
	var configErr error

	if supportsWireguard {
		// A wireguard connect call needs to generate a wireguard key and add it to the config
		// Also the server could send back an OpenVPN config if it supports OpenVPN
		config, configType, configErr = wireguardGetConfig(server, supportsOpenVPN)
	} else {
		config, configType, configErr = openVPNGetConfig(server)
	}

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}

	return config, configType, nil
}

func askForProfileID(server Server) error {
	errorMessage := "failed asking for a server profile ID"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	if !base.FSM.HasTransition(fsm.ASK_PROFILE) {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: fsm.WrongStateTransitionError{Got: base.FSM.Current, Want: fsm.ASK_PROFILE}.CustomError()}
	}
	base.FSM.GoTransitionWithData(fsm.ASK_PROFILE, base.ProfilesRaw, false)
	return nil
}

type ServerInfoScreen struct {
	Identifier     string            `json:"identifier"`
	DisplayName    map[string]string `json:"display_name"`
	CountryCode    string            `json:"country_code,omitempty"`
	SupportContact []string          `json:"support_contact"`
	ProfilesRaw    string            `json:"profiles"`
	ExpireTime     int64             `json:"expire_time"`
	Type           string            `json:"server_type"`
}

func (servers *Servers) GetCurrentServerInfoJSON() (string, error) {
	errorMessage := "failed getting JSON for server"

	currentServer, currentServerErr := servers.GetCurrentServer()
	if currentServerErr != nil {
		return "{}", &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
	}

	serverInfoScreen := &ServerInfoScreen{}

	base, baseErr := currentServer.GetBase()

	if baseErr != nil {
		return "{}", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	serverInfoScreen.Identifier = base.URL
	serverInfoScreen.DisplayName = base.DisplayName
	serverInfoScreen.SupportContact = base.SupportContact
	serverInfoScreen.ProfilesRaw = base.ProfilesRaw
	serverInfoScreen.ExpireTime = base.EndTime
	serverInfoScreen.Type = base.Type

	if servers.IsSecureInternet {
		serverInfoScreen.Identifier = servers.SecureInternetHomeServer.HomeOrganizationID
		serverInfoScreen.CountryCode = servers.SecureInternetHomeServer.CurrentLocation
	}

	bytes, bytesErr := json.Marshal(serverInfoScreen)

	if bytesErr != nil {
		return "{}", &types.WrappedErrorMessage{Message: errorMessage, Err: bytesErr}
	}

	return string(bytes), nil
}

func GetConfig(server Server, forceTCP bool) (string, string, error) {
	errorMessage := "failed getting an OpenVPN/WireGuard configuration"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	if !base.FSM.InState(fsm.REQUEST_CONFIG) {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: fsm.WrongStateError{Got: base.FSM.Current, Want: fsm.REQUEST_CONFIG}.CustomError()}
	}

	// Get new profiles using the info call
	// This does not override the current profile
	infoErr := APIInfo(server)
	if infoErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: infoErr}
	}

	// If there was a profile chosen and it doesn't exist anymore, reset it
	if base.Profiles.Current != "" {
		_, existsProfileErr := getCurrentProfile(server)
		if existsProfileErr != nil {
			base.Logger.Log(log.LOG_INFO, fmt.Sprintf("Profile %s no longer exists, resetting the profile", base.Profiles.Current))
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
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: profileErr}
	}

	return getConfigWithProfile(server, forceTCP)
}

type ServerGetCurrentProfileNotFoundError struct {
	ProfileID string
}

func (e *ServerGetCurrentProfileNotFoundError) Error() string {
	return fmt.Sprintf("failed to get current profile, profile with ID: %s not found", e.ProfileID)
}

type ServerGetConfigForceTCPError struct{}

func (e *ServerGetConfigForceTCPError) Error() string {
	return fmt.Sprintf("failed to get config, force TCP is on but the server does not support OpenVPN")
}

type ServerGetSecureInternetHomeError struct{}

func (e *ServerGetSecureInternetHomeError) Error() string {
	return "failed to get secure internet home server, not found"
}

type ServerEnsureServerEmptyURLError struct{}

func (e *ServerEnsureServerEmptyURLError) Error() string {
	return "failed ensuring server, empty url provided"
}

type ServerGetCurrentNoMapError struct{}

func (e *ServerGetCurrentNoMapError) Error() string {
	return "failed getting current server, no servers available"
}

type ServerGetCurrentNotFoundError struct{}

func (e *ServerGetCurrentNotFoundError) Error() string {
	return "failed getting current server, not found"
}

type ServerSecureInternetMapNotFoundError struct{}

func (e *ServerSecureInternetMapNotFoundError) Error() string {
	return "secure internet map not found"
}

type ServerSecureInternetBaseNotFoundError struct {
	Current string
}

func (e *ServerSecureInternetBaseNotFoundError) Error() string {
	return fmt.Sprintf("secure internet base not found with current location: %s", e.Current)
}
