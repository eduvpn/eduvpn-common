package server

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

// A secure internet server which has its own OAuth tokens
// It specifies the current location url it is connected to.
type SecureInternetHomeServer struct {
	Auth        oauth.OAuth       `json:"oauth"`
	DisplayName map[string]string `json:"display_name"`

	// The home server has a list of info for each configured server location
	BaseMap map[string]*Base `json:"base_map"`

	// We have the authorization URL template, the home organization ID and the current location
	AuthorizationTemplate string `json:"authorization_template"`
	HomeOrganizationID    string `json:"home_organization_id"`
	CurrentLocation       string `json:"current_location"`
}

func (servers *Servers) GetSecureInternetHomeServer() (*SecureInternetHomeServer, error) {
	if !servers.HasSecureLocation() {
		return nil, errors.New("no secure internet home server")
	}
	return &servers.SecureInternetHomeServer, nil
}

func (servers *Servers) SetSecureInternet(server Server) error {
	errorMessage := "failed setting secure internet server"
	base, baseErr := server.Base()
	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	if base.Type != "secure_internet" {
		return types.NewWrappedError(errorMessage, errors.New("not a secure internet server"))
	}

	// The location should already be configured
	// TODO: check for location?
	servers.IsType = SecureInternetServerType
	return nil
}

func (servers *Servers) RemoveSecureInternet() {
	// Empty out the struct
	servers.SecureInternetHomeServer = SecureInternetHomeServer{}

	// If the current server is secure internet, default to custom server
	if servers.IsType == SecureInternetServerType {
		servers.IsType = CustomServerType
	}
}

func (server *SecureInternetHomeServer) TemplateAuth() func(string) string {
	return func(authURL string) string {
		return util.ReplaceWAYF(server.AuthorizationTemplate, authURL, server.HomeOrganizationID)
	}
}

func (server *SecureInternetHomeServer) Base() (*Base, error) {
	errorMessage := "failed getting current secure internet home base"
	if server.BaseMap == nil {
		return nil, types.NewWrappedError(
			errorMessage,
			&SecureInternetMapNotFoundError{},
		)
	}

	base, exists := server.BaseMap[server.CurrentLocation]

	if !exists {
		return nil, types.NewWrappedError(
			errorMessage,
			&SecureInternetBaseNotFoundError{Current: server.CurrentLocation},
		)
	}
	return base, nil
}

func (server *SecureInternetHomeServer) OAuth() *oauth.OAuth {
	return &server.Auth
}

func (servers *Servers) HasSecureLocation() bool {
	return servers.SecureInternetHomeServer.CurrentLocation != ""
}

func (server *SecureInternetHomeServer) addLocation(
	locationServer *types.DiscoveryServer,
) (*Base, error) {
	errorMessage := "failed adding a location"
	// Initialize the base map if it is non-nil
	if server.BaseMap == nil {
		server.BaseMap = make(map[string]*Base)
	}

	// Add the location to the base map
	base, exists := server.BaseMap[locationServer.CountryCode]

	if !exists || base == nil {
		// Create the base to be added to the map
		base = &Base{}
		base.URL = locationServer.BaseURL
		base.DisplayName = server.DisplayName
		base.SupportContact = locationServer.SupportContact
		base.Type = "secure_internet"
		endpointsErr := base.InitializeEndpoints()
		if endpointsErr != nil {
			return nil, types.NewWrappedError(errorMessage, endpointsErr)
		}
	}

	// Ensure it is in the map
	server.BaseMap[locationServer.CountryCode] = base
	return base, nil
}

// Initializes the home server and adds its own location.
func (server *SecureInternetHomeServer) init(
	homeOrg *types.DiscoveryOrganization,
	homeLocation *types.DiscoveryServer,
) error {
	errorMessage := "failed initializing secure internet home server"

	if server.HomeOrganizationID != homeOrg.OrgID {
		// New home organisation, clear everything
		*server = SecureInternetHomeServer{}
	}

	// Make sure to set the organization ID
	server.HomeOrganizationID = homeOrg.OrgID
	server.DisplayName = homeOrg.DisplayName

	// Make sure to set the authorization URL template
	server.AuthorizationTemplate = homeLocation.AuthenticationURLTemplate

	base, baseErr := server.addLocation(homeLocation)

	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	// Make sure oauth contains our endpoints
	server.Auth.Init(base.URL, base.Endpoints.API.V3.Authorization, base.Endpoints.API.V3.Token)
	return nil
}

type SecureInternetHomeNotFoundError struct{}

func (e *SecureInternetHomeNotFoundError) Error() string {
	return "failed to get secure internet home server, not found"
}

type SecureInternetMapNotFoundError struct{}

func (e *SecureInternetMapNotFoundError) Error() string {
	return "secure internet map not found"
}

type SecureInternetBaseNotFoundError struct {
	Current string
}

func (e *SecureInternetBaseNotFoundError) Error() string {
	return fmt.Sprintf("secure internet base not found with current location: %s", e.Current)
}
