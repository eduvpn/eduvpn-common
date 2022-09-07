package server

import (
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/oauth"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
)

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

func (servers *Servers) RemoveSecureInternet() {
	// Empty out the struct
	servers.SecureInternetHomeServer = SecureInternetHomeServer{}

	// If the current server is secure internet, default to custom server
	if servers.IsType == SecureInternetServerType {
		servers.IsType = CustomServerType
	}
}

func (secure *SecureInternetHomeServer) GetOAuth() *oauth.OAuth {
	return &secure.OAuth
}

func (secure *SecureInternetHomeServer) GetTemplateAuth() func(string) string {
	return func(authURL string) string {
		return util.ReplaceWAYF(secure.AuthorizationTemplate, authURL, secure.HomeOrganizationID)
	}
}

func (server *SecureInternetHomeServer) GetBase() (*ServerBase, error) {
	errorMessage := "failed getting current secure internet home base"
	if server.BaseMap == nil {
		return nil, &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     &ServerSecureInternetMapNotFoundError{},
		}
	}

	base, exists := server.BaseMap[server.CurrentLocation]

	if !exists {
		return nil, &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     &ServerSecureInternetBaseNotFoundError{Current: server.CurrentLocation},
		}
	}
	return base, nil
}

func (servers *Servers) HasSecureLocation() bool {
	return servers.SecureInternetHomeServer.CurrentLocation != ""
}

func (secure *SecureInternetHomeServer) addLocation(
	locationServer *types.DiscoveryServer,
) (*ServerBase, error) {
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

	// Ensure it is in the map
	secure.BaseMap[locationServer.CountryCode] = base
	return base, nil
}

// Initializes the home server and adds its own location
func (secure *SecureInternetHomeServer) init(
	homeOrg *types.DiscoveryOrganization,
	homeLocation *types.DiscoveryServer,
) error {
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

	base, baseErr := secure.addLocation(homeLocation)

	if baseErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	// Make sure oauth contains our endpoints
	secure.OAuth.Init(base.Endpoints.API.V3.Authorization, base.Endpoints.API.V3.Token)
	return nil
}

type ServerGetSecureInternetHomeError struct{}

func (e *ServerGetSecureInternetHomeError) Error() string {
	return "failed to get secure internet home server, not found"
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
