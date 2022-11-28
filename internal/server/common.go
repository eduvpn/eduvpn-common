package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/wireguard"
	"github.com/eduvpn/eduvpn-common/types"
)

// The base type for servers.
type Base struct {
	URL            string            `json:"base_url"`
	DisplayName    map[string]string `json:"display_name"`
	SupportContact []string          `json:"support_contact"`
	Endpoints      Endpoints   `json:"endpoints"`
	Profiles       ProfileInfo `json:"profiles"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"expire_time"`
	Type           string            `json:"server_type"`
}

type Type int8

const (
	CustomServerType Type = iota
	InstituteAccessServerType
	SecureInternetServerType
)

type Servers struct {
	// A custom server is just an institute access server under the hood
	CustomServers            InstituteAccessServers   `json:"custom_servers"`
	InstituteServers         InstituteAccessServers   `json:"institute_servers"`
	SecureInternetHomeServer SecureInternetHomeServer `json:"secure_internet_home"`
	IsType                   Type               `json:"is_secure_internet"`
}

type Server interface {
	OAuth() *oauth.OAuth

	// Get the authorization URL template function
	TemplateAuth() func(string) string

	// Gets the server base
	Base() (*Base, error)
}

type Profile struct {
	ID             string   `json:"profile_id"`
	DisplayName    string   `json:"display_name"`
	VPNProtoList   []string `json:"vpn_proto_list"`
	DefaultGateway bool     `json:"default_gateway"`
}

type ProfileListInfo struct {
	ProfileList []Profile `json:"profile_list"`
}

type ProfileInfo struct {
	Current string `json:"current_profile"`
	Info    ProfileListInfo `json:"info"`
}

func (info ProfileInfo) GetCurrentProfileIndex() int {
	index := 0
	for _, profile := range info.Info.ProfileList {
		if profile.ID == info.Current {
			return index
		}
		index++
	}
	// Default is 'first' profile
	return 0
}

type EndpointList struct {
	API           string `json:"api_endpoint"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
}

// Struct that defines the json format for /.well-known/vpn-user-portal".
type Endpoints struct {
	API struct {
		V2 EndpointList `json:"http://eduvpn.org/api#2"`
		V3 EndpointList `json:"http://eduvpn.org/api#3"`
	} `json:"api"`
	V string `json:"v"`
}

func (servers *Servers) GetCurrentServer() (Server, error) {
	errorMessage := "failed getting current server"
	if servers.IsType == SecureInternetServerType {
		if !servers.HasSecureLocation() {
			return nil, types.NewWrappedError(
				errorMessage,
				&CurrentNotFoundError{},
			)
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
		return nil, types.NewWrappedError(
			errorMessage,
			&CurrentNoMapError{},
		)
	}
	server, exists := bases[currentServerURL]

	if !exists || server == nil {
		return nil, types.NewWrappedError(
			errorMessage,
			&CurrentNotFoundError{},
		)
	}
	return server, nil
}

func (servers *Servers) addInstituteAndCustom(
	discoServer *types.DiscoveryServer,
	isCustom bool,
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

	instituteInitErr := server.init(
		url,
		discoServer.DisplayName,
		discoServer.Type,
		discoServer.SupportContact,
	)
	if instituteInitErr != nil {
		return nil, types.NewWrappedError(errorMessage, instituteInitErr)
	}
	toAddServers.Map[url] = server
	servers.IsType = serverType
	return server, nil
}

func (servers *Servers) AddInstituteAccessServer(
	instituteServer *types.DiscoveryServer,
) (Server, error) {
	return servers.addInstituteAndCustom(instituteServer, false)
}

func (servers *Servers) AddCustomServer(
	customServer *types.DiscoveryServer,
) (Server, error) {
	return servers.addInstituteAndCustom(customServer, true)
}

func (servers *Servers) GetSecureLocation() string {
	return servers.SecureInternetHomeServer.CurrentLocation
}

func (servers *Servers) SetSecureLocation(
	chosenLocationServer *types.DiscoveryServer,
) error {
	errorMessage := "failed to set secure location"
	// Make sure to add the current location
	_, addLocationErr := servers.SecureInternetHomeServer.addLocation(chosenLocationServer)

	if addLocationErr != nil {
		return types.NewWrappedError(errorMessage, addLocationErr)
	}

	servers.SecureInternetHomeServer.CurrentLocation = chosenLocationServer.CountryCode
	return nil
}

func (servers *Servers) AddSecureInternet(
	secureOrg *types.DiscoveryOrganization,
	secureServer *types.DiscoveryServer,
) (Server, error) {
	errorMessage := "failed adding secure internet server"
	// If we have specified an organization ID
	// We also need to get an authorization template
	initErr := servers.SecureInternetHomeServer.init(secureOrg, secureServer)

	if initErr != nil {
		return nil, types.NewWrappedError(errorMessage, initErr)
	}

	servers.IsType = SecureInternetServerType
	return &servers.SecureInternetHomeServer, nil
}

func ShouldRenewButton(server Server) bool {
	base, baseErr := server.Base()

	if baseErr != nil {
		// FIXME: Log error here?
		return false
	}

	// Get current time
	current := time.Now()

	// Session is expired
	if !current.Before(base.EndTime) {
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

func OAuthURL(server Server, name string) (string, error) {
	return server.OAuth().AuthURL(name, server.TemplateAuth())
}

func OAuthExchange(server Server) error {
	return server.OAuth().Exchange()
}

func HeaderToken(server Server) (string, error) {
	token, tokenErr := server.OAuth().AccessToken()
	if tokenErr != nil {
		return "", types.NewWrappedError("failed getting server token for HTTP Header", tokenErr)
	}
	return token, nil
}

func MarkTokenExpired(server Server) {
	server.OAuth().SetTokenExpired()
}

func MarkTokensForRenew(server Server) {
	server.OAuth().SetTokenRenew()
}

func NeedsRelogin(server Server) bool {
	_, tokenErr := HeaderToken(server)
	return tokenErr != nil
}

func CancelOAuth(server Server) {
	server.OAuth().Cancel()
}

func (profile *Profile) supportsProtocol(protocol string) bool {
	for _, proto := range profile.VPNProtoList {
		if proto == protocol {
			return true
		}
	}
	return false
}

func (profile *Profile) supportsWireguard() bool {
	return profile.supportsProtocol("wireguard")
}

func (profile *Profile) supportsOpenVPN() bool {
	return profile.supportsProtocol("openvpn")
}

func CurrentProfile(server Server) (*Profile, error) {
	errorMessage := "failed getting current profile"
	base, baseErr := server.Base()

	if baseErr != nil {
		return nil, types.NewWrappedError(errorMessage, baseErr)
	}
	profileID := base.Profiles.Current
	for _, profile := range base.Profiles.Info.ProfileList {
		if profile.ID == profileID {
			return &profile, nil
		}
	}

	return nil, types.NewWrappedError(
		errorMessage,
		&CurrentProfileNotFoundError{ProfileID: profileID},
	)
}

func (base *Base) InitializeEndpoints() error {
	errorMessage := "failed initializing endpoints"
	endpoints, endpointsErr := APIGetEndpoints(base.URL)
	if endpointsErr != nil {
		return types.NewWrappedError(errorMessage, endpointsErr)
	}
	base.Endpoints = *endpoints
	return nil
}

func (base *Base) ValidProfiles(clientSupportsWireguard bool) ProfileInfo {
	var validProfiles []Profile
	for _, profile := range base.Profiles.Info.ProfileList {
		// Not a valid profile because it does not support openvpn
		// Also the client does not support wireguard
		if !profile.supportsOpenVPN() && !clientSupportsWireguard {
			continue
		}
		validProfiles = append(validProfiles, profile)
	}
	return ProfileInfo{Current: base.Profiles.Current, Info: ProfileListInfo{ProfileList: validProfiles}}
}

func ValidProfiles(server Server, clientSupportsWireguard bool) (*ProfileInfo, error) {
	errorMessage := "failed to get valid profiles"
	// No error wrapping here otherwise we wrap it too much
	base, baseErr := server.Base()
	if baseErr != nil {
		return nil, types.NewWrappedError(errorMessage, baseErr)
	}
	profiles := base.ValidProfiles(clientSupportsWireguard)
	if len(profiles.Info.ProfileList) == 0 {
		return nil, types.NewWrappedError(errorMessage, errors.New("no profiles found with supported protocols"))
	}
	return &profiles, nil
}

func wireguardGetConfig(server Server, preferTCP bool, supportsOpenVPN bool) (string, string, error) {
	errorMessage := "failed getting server WireGuard configuration"
	base, baseErr := server.Base()

	if baseErr != nil {
		return "", "", types.NewWrappedError(errorMessage, baseErr)
	}

	profileID := base.Profiles.Current
	wireguardKey, wireguardErr := wireguard.GenerateKey()

	if wireguardErr != nil {
		return "", "", types.NewWrappedError(errorMessage, wireguardErr)
	}

	wireguardPublicKey := wireguardKey.PublicKey().String()
	config, content, expires, configErr := APIConnectWireguard(
		server,
		profileID,
		wireguardPublicKey,
		preferTCP,
		supportsOpenVPN,
	)

	if configErr != nil {
		return "", "", types.NewWrappedError(errorMessage, configErr)
	}

	// Store start and end time
	base.StartTime = time.Now()
	base.EndTime = expires

	if content == "wireguard" {
		// This needs the go code a way to identify a connection
		// Use the uuid of the connection e.g. on Linux
		// This needs the client code to call the go code

		config = wireguard.ConfigAddKey(config, wireguardKey)
	}

	return config, content, nil
}

func openVPNGetConfig(server Server, preferTCP bool) (string, string, error) {
	errorMessage := "failed getting server OpenVPN configuration"
	base, baseErr := server.Base()

	if baseErr != nil {
		return "", "", types.NewWrappedError(errorMessage, baseErr)
	}
	profileID := base.Profiles.Current
	configOpenVPN, expires, configErr := APIConnectOpenVPN(server, profileID, preferTCP)

	// Store start and end time
	base.StartTime = time.Now()
	base.EndTime = expires

	if configErr != nil {
		return "", "", types.NewWrappedError(errorMessage, configErr)
	}

	return configOpenVPN, "openvpn", nil
}

func HasValidProfile(server Server, clientSupportsWireguard bool) (bool, error) {
	errorMessage := "failed has valid profile check"

	// Get new profiles using the info call
	// This does not override the current profile
	infoErr := APIInfo(server)
	if infoErr != nil {
		return false, types.NewWrappedError(errorMessage, infoErr)
	}

	base, baseErr := server.Base()
	if baseErr != nil {
		return false, types.NewWrappedError(errorMessage, baseErr)
	}

	// If there was a profile chosen and it doesn't exist anymore, reset it
	if base.Profiles.Current != "" {
		_, existsProfileErr := CurrentProfile(server)
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
		profile, profileErr := CurrentProfile(server)
		// shouldn't happen
		if profileErr != nil {
			return false, types.NewWrappedError(errorMessage, profileErr)
		}
		// Profile does not support OpenVPN but the client also doesn't support WireGuard
		if !profile.supportsOpenVPN() && !clientSupportsWireguard {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

func RefreshEndpoints(server Server) error {
	errorMessage := "failed to refresh server endpoints"

	// Re-initialize the endpoints
	// TODO: Make this a warning instead?
	base, baseErr := server.Base()
	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	endpointsErr := base.InitializeEndpoints()
	if endpointsErr != nil {
		return types.NewWrappedError(errorMessage, endpointsErr)
	}

	return nil
}

func Config(server Server, clientSupportsWireguard bool, preferTCP bool) (string, string, error) {
	errorMessage := "failed getting an OpenVPN/WireGuard configuration"

	profile, profileErr := CurrentProfile(server)
	if profileErr != nil {
		return "", "", types.NewWrappedError(errorMessage, profileErr)
	}

	supportsOpenVPN := profile.supportsOpenVPN()
	supportsWireguard := profile.supportsWireguard() && clientSupportsWireguard

	var config string
	var configType string
	var configErr error

	// The config supports wireguard, do a specialized request with a public key
	if supportsWireguard {
		// A wireguard connect call needs to generate a wireguard key and add it to the config
		// Also the server could send back an OpenVPN config if it supports OpenVPN
		config, configType, configErr = wireguardGetConfig(server, preferTCP, supportsOpenVPN)
	//  The config only supports OpenVPN
	} else if supportsOpenVPN {
		config, configType, configErr = openVPNGetConfig(server, preferTCP)
	// The config supports no available protocol because the profile only supports WireGuard but the client doesn't
	} else {
		return "", "", types.NewWrappedError(errorMessage, errors.New("no supported protocol found"))
	}

	if configErr != nil {
		return "", "", types.NewWrappedError(errorMessage, configErr)
	}

	return config, configType, nil
}

func Disconnect(server Server) {
	APIDisconnect(server)
}

type CurrentProfileNotFoundError struct {
	ProfileID string
}

func (e *CurrentProfileNotFoundError) Error() string {
	return fmt.Sprintf("failed to get current profile, profile with ID: %s not found", e.ProfileID)
}

type ConfigPreferTCPError struct{}

func (e *ConfigPreferTCPError) Error() string {
	return "failed to get config, prefer TCP is on but the server does not support OpenVPN"
}

type EmptyURLError struct{}

func (e *EmptyURLError) Error() string {
	return "failed ensuring server, empty url provided"
}

type CurrentNoMapError struct{}

func (e *CurrentNoMapError) Error() string {
	return "failed getting current server, no servers available"
}

type CurrentNotFoundError struct{}

func (e *CurrentNotFoundError) Error() string {
	return "failed getting current server, not found"
}
