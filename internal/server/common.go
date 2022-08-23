package server

import (
	"fmt"
	"time"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
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
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"expire_time"`
	Type           string            `json:"server_type"`
	FSM            *fsm.FSM          `json:"-"`
}

type ServerType int8

const (
	CustomServerType ServerType = iota
	InstituteAccessServerType
	SecureInternetServerType
)

type Servers struct {
	// A custom server is just an institute access server under the hood
	CustomServers            InstituteAccessServers   `json:"custom_servers"`
	InstituteServers         InstituteAccessServers   `json:"institute_servers"`
	SecureInternetHomeServer SecureInternetHomeServer `json:"secure_internet_home"`
	IsType                   ServerType               `json:"is_secure_internet"`
}

type Server interface {
	// Gets the current OAuth object
	GetOAuth() *oauth.OAuth

	// Get the authorization URL template function
	GetTemplateAuth() func(string) string

	// Gets the server base
	GetBase() (*ServerBase, error)
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

func (servers *Servers) GetCurrentServer() (Server, error) {
	errorMessage := "failed getting current server"
	if servers.IsType == SecureInternetServerType {
		if !servers.HasSecureLocation() {
			return nil, &types.WrappedErrorMessage{
				Message: errorMessage,
				Err:     &ServerGetCurrentNotFoundError{},
			}
		}
		return &servers.SecureInternetHomeServer, nil
	}

	serversStruct := &servers.InstituteServers

	if servers.IsType == CustomServerType {
		serversStruct = &servers.CustomServers
	}
	currentServerURL := serversStruct.CurrentURL
	bases := serversStruct.Map
	if bases == nil {
		return nil, &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     &ServerGetCurrentNoMapError{},
		}
	}
	server, exists := bases[currentServerURL]

	if !exists || server == nil {
		return nil, &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     &ServerGetCurrentNotFoundError{},
		}
	}
	return server, nil
}

type ServersConfiguredScreen struct {
	CustomServers          []ServerInfoScreen `json:"custom_servers"`
	InstituteAccessServers []ServerInfoScreen `json:"institute_access_servers"`
	SecureInternetServer   *ServerInfoScreen  `json:"secure_internet_server"`
}

type ServerInfoScreen struct {
	Identifier     string            `json:"identifier"`
	DisplayName    map[string]string `json:"display_name"`
	CountryCode    string            `json:"country_code,omitempty"`
	SupportContact []string          `json:"support_contact"`
	Profiles       ServerProfileInfo `json:"profiles"`
	ExpireTime     int64             `json:"expire_time"`
	Type           string            `json:"server_type"`
}

func getServerInfoScreen(base ServerBase) ServerInfoScreen {
	serverInfoScreen := ServerInfoScreen{}
	serverInfoScreen.Identifier = base.URL
	serverInfoScreen.DisplayName = base.DisplayName
	serverInfoScreen.SupportContact = base.SupportContact
	serverInfoScreen.Profiles = base.Profiles

	// If we still have the default end time, return 0
	// Such that clients will still be able to parse it correctly
	if base.EndTime.IsZero() {
		serverInfoScreen.ExpireTime = 0
	} else {
		serverInfoScreen.ExpireTime = base.EndTime.Unix()
	}
	serverInfoScreen.Type = base.Type

	return serverInfoScreen
}

func (servers *Servers) GetServersConfigured() *ServersConfiguredScreen {
	customServersInfo := []ServerInfoScreen{}
	instituteServersInfo := []ServerInfoScreen{}
	var secureInternetServerInfo *ServerInfoScreen = nil

	for _, server := range servers.CustomServers.Map {
		serverInfoScreen := getServerInfoScreen(server.Base)
		customServersInfo = append(customServersInfo, serverInfoScreen)
	}

	for _, server := range servers.InstituteServers.Map {
		serverInfoScreen := getServerInfoScreen(server.Base)
		instituteServersInfo = append(instituteServersInfo, serverInfoScreen)
	}

	secureInternetBase, secureInternetBaseErr := servers.SecureInternetHomeServer.GetBase()

	if secureInternetBaseErr == nil && secureInternetBase != nil {
		// FIXME: log error?
		secureInternetServerInfoReturned := getServerInfoScreen(*secureInternetBase)
		secureInternetServerInfo = &secureInternetServerInfoReturned
		secureInternetServerInfo.Identifier = servers.SecureInternetHomeServer.HomeOrganizationID
		secureInternetServerInfo.CountryCode = servers.SecureInternetHomeServer.CurrentLocation
	}

	return &ServersConfiguredScreen{
		CustomServers:          customServersInfo,
		InstituteAccessServers: instituteServersInfo,
		SecureInternetServer:   secureInternetServerInfo,
	}
}

func (servers *Servers) GetCurrentServerInfo() (*ServerInfoScreen, error) {
	errorMessage := "failed getting current server info"

	currentServer, currentServerErr := servers.GetCurrentServer()
	if currentServerErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
	}

	base, baseErr := currentServer.GetBase()

	if baseErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	serverInfoScreen := getServerInfoScreen(*base)

	if servers.IsType == SecureInternetServerType {
		serverInfoScreen.Identifier = servers.SecureInternetHomeServer.HomeOrganizationID
		serverInfoScreen.CountryCode = servers.SecureInternetHomeServer.CurrentLocation
	}

	return &serverInfoScreen, nil
}

func (servers *Servers) addInstituteAndCustom(
	discoServer *types.DiscoveryServer,
	isCustom bool,
	fsm *fsm.FSM,
) (Server, error) {
	url := discoServer.BaseURL
	errorMessage := fmt.Sprintf("failed adding institute access server: %s", url)
	toAddServers := &servers.InstituteServers
	serverType := InstituteAccessServerType

	if isCustom {
		toAddServers = &servers.CustomServers
		serverType = CustomServerType
	}

	if toAddServers.Map == nil {
		toAddServers.Map = make(map[string]*InstituteAccessServer)
	}

	server, exists := toAddServers.Map[url]

	// initialize the server if it doesn't exist yet
	if !exists {
		server = &InstituteAccessServer{}
	}

	// Set the current server
	toAddServers.CurrentURL = url
	instituteInitErr := server.init(
		url,
		discoServer.DisplayName,
		discoServer.Type,
		discoServer.SupportContact,
		fsm,
	)
	if instituteInitErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: instituteInitErr}
	}
	toAddServers.Map[url] = server
	servers.IsType = serverType
	return server, nil
}

func (servers *Servers) AddInstituteAccessServer(
	instituteServer *types.DiscoveryServer,
	fsm *fsm.FSM,
) (Server, error) {
	return servers.addInstituteAndCustom(instituteServer, false, fsm)
}

func (servers *Servers) AddCustomServer(
	customServer *types.DiscoveryServer,
	fsm *fsm.FSM,
) (Server, error) {
	return servers.addInstituteAndCustom(customServer, true, fsm)
}

func (servers *Servers) GetSecureLocation() string {
	return servers.SecureInternetHomeServer.CurrentLocation
}

func (servers *Servers) SetSecureLocation(
	chosenLocationServer *types.DiscoveryServer,
	fsm *fsm.FSM,
) error {
	errorMessage := "failed to set secure location"
	// Make sure to add the current location
	_, addLocationErr := servers.SecureInternetHomeServer.addLocation(chosenLocationServer, fsm)

	if addLocationErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: addLocationErr}
	}

	servers.SecureInternetHomeServer.CurrentLocation = chosenLocationServer.CountryCode
	return nil
}

func (servers *Servers) AddSecureInternet(
	secureOrg *types.DiscoveryOrganization,
	secureServer *types.DiscoveryServer,
	fsm *fsm.FSM,
) (Server, error) {
	errorMessage := "failed adding secure internet server"
	// If we have specified an organization ID
	// We also need to get an authorization template
	initErr := servers.SecureInternetHomeServer.init(secureOrg, secureServer, fsm)

	if initErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: initErr}
	}

	servers.IsType = SecureInternetServerType
	return &servers.SecureInternetHomeServer, nil
}

func ShouldRenewButton(server Server) bool {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		// FIXME: Log error here?
		return false
	}

	// Get current time
	current := util.GetCurrentTime()

	// Session is expired
	if current.After(base.EndTime) {
		return true
	}

	// 30 minutes have not passed
	if !current.After(base.StartTime.Add(30 * time.Minute)) {
		return false
	}

	// Session will not expire today
	if !current.Add(24 * time.Hour).After(base.EndTime) {
		return false
	}

	// Session duration is less than 24 hours but not 75% has passed
	duration := base.EndTime.Sub(base.StartTime)
	percentTime := base.StartTime.Add((duration / 4) * 3)
	if duration < time.Duration(24*time.Hour) && !current.After(percentTime) {
		return false
	}

	return true
}

func Login(server Server) error {
	return server.GetOAuth().Login("org.eduvpn.app.linux", server.GetTemplateAuth())
}

func EnsureTokens(server Server) error {
	errorMessage := "failed ensuring server tokens"
	if server.GetOAuth().NeedsRelogin() {
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

	return nil, &types.WrappedErrorMessage{
		Message: errorMessage,
		Err:     &ServerGetCurrentProfileNotFoundError{ProfileID: profileID},
	}
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
	config, content, expires, configErr := APIConnectWireguard(
		server,
		profile_id,
		wireguardPublicKey,
		supportsOpenVPN,
	)

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}

	// Store start and end time
	base.StartTime = util.GetCurrentTime()
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
	base.StartTime = util.GetCurrentTime()
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
		return "", "", &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: fsm.WrongStateTransitionError{
				Got:  base.FSM.Current,
				Want: fsm.HAS_CONFIG,
			}.CustomError(),
		}
	}
	profile, profileErr := getCurrentProfile(server)

	if profileErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: profileErr}
	}

	supportsOpenVPN := profile.supportsOpenVPN()
	supportsWireguard := profile.supportsWireguard()

	// If forceTCP we must be able to get a config with OpenVPN
	if forceTCP && supportsOpenVPN {
		return "", "", &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     &ServerGetConfigForceTCPError{},
		}
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
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: fsm.WrongStateTransitionError{
				Got:  base.FSM.Current,
				Want: fsm.ASK_PROFILE,
			}.CustomError(),
		}
	}
	base.FSM.GoTransitionWithData(fsm.ASK_PROFILE, &base.Profiles, false)
	return nil
}

func GetConfig(server Server, forceTCP bool) (string, string, error) {
	errorMessage := "failed getting an OpenVPN/WireGuard configuration"
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	if !base.FSM.InState(fsm.REQUEST_CONFIG) {
		return "", "", &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: fsm.WrongStateError{
				Got:  base.FSM.Current,
				Want: fsm.REQUEST_CONFIG,
			}.CustomError(),
		}
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

func Disconnect(server Server) {
	APIDisconnect(server)
}

type ServerGetCurrentProfileNotFoundError struct {
	ProfileID string
}

func (e *ServerGetCurrentProfileNotFoundError) Error() string {
	return fmt.Sprintf("failed to get current profile, profile with ID: %s not found", e.ProfileID)
}

type ServerGetConfigForceTCPError struct{}

func (e *ServerGetConfigForceTCPError) Error() string {
	return fmt.Sprintf(
		"failed to get config, force TCP is on but the server does not support OpenVPN",
	)
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
