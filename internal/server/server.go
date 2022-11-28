package server

import (
	"errors"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/wireguard"
	"github.com/eduvpn/eduvpn-common/types"
)

type Type int8

const (
	CustomServerType Type = iota
	InstituteAccessServerType
	SecureInternetServerType
)

type Server interface {
	OAuth() *oauth.OAuth

	// Get the authorization URL template function
	TemplateAuth() func(string) string

	// Gets the server base
	Base() (*Base, error)
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

func ValidProfiles(server Server, clientSupportsWireguard bool) (*ProfileInfo, error) {
	errorMessage := "failed to get valid profiles"
	// No error wrapping here otherwise we wrap it too much
	base, baseErr := server.Base()
	if baseErr != nil {
		return nil, types.NewWrappedError(errorMessage, baseErr)
	}
	profiles := base.ValidProfiles(clientSupportsWireguard)
	if len(profiles.Info.ProfileList) == 0 {
		return nil, types.NewWrappedError(
			errorMessage,
			errors.New("no profiles found with supported protocols"),
		)
	}
	return &profiles, nil
}

func wireguardGetConfig(
	server Server,
	preferTCP bool,
	supportsOpenVPN bool,
) (string, string, error) {
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

	switch {
	// The config supports wireguard and optionally openvpn
	case supportsWireguard:
		// A wireguard connect call needs to generate a wireguard key and add it to the config
		// Also the server could send back an OpenVPN config if it supports OpenVPN
		config, configType, configErr = wireguardGetConfig(server, preferTCP, supportsOpenVPN)
	//  The config only supports OpenVPN
	case supportsOpenVPN:
		config, configType, configErr = openVPNGetConfig(server, preferTCP)
		// The config supports no available protocol because the profile only supports WireGuard but the client doesn't
	default:
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
