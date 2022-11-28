// package discovery implements the server discovery by contacting disco.eduvpn.org and returning the data as a Go structure
package discovery

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/verify"
	"github.com/eduvpn/eduvpn-common/types"
)

// Discovery is the main structure used for this package.
type Discovery struct {
	// organizations represents the organizations that are returned by the discovery server
	organizations types.DiscoveryOrganizations

	// servers represents the servers that are returned by the discovery server
	servers types.DiscoveryServers
}

// discoFile is a helper function that gets a disco JSON and fills the structure with it
// If it was unsuccessful it returns an error.
func discoFile(jsonFile string, previousVersion uint64, structure interface{}) error {
	errorMessage := fmt.Sprintf("failed getting file: %s from the Discovery server", jsonFile)
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	_, fileBody, fileErr := http.Get(fileURL)

	if fileErr != nil {
		return types.NewWrappedError(errorMessage, fileErr)
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	_, sigBody, sigFileErr := http.Get(sigURL)

	if sigFileErr != nil {
		return types.NewWrappedError(errorMessage, sigFileErr)
	}

	// Verify signature
	// Set this to true when we want to force prehash
	forcePrehash := false
	verifySuccess, verifyErr := verify.Verify(
		string(sigBody),
		fileBody,
		jsonFile,
		previousVersion,
		forcePrehash,
	)

	if !verifySuccess || verifyErr != nil {
		return types.NewWrappedError(errorMessage, verifyErr)
	}

	// Parse JSON to extract version and list
	jsonErr := json.Unmarshal(fileBody, structure)

	if jsonErr != nil {
		return types.NewWrappedError(errorMessage, jsonErr)
	}

	return nil
}

// DetermineOrganizationsUpdate returns a boolean indicating whether or not the discovery organizations should be updated
// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [TODO] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [TODO] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
// - [IMPLEMENTED using a custom error message] NOTE: when the org_id that the user chose previously is no longer available in organization_list.json the application should ask the user to choose their organization (again). This can occur for example when the organization replaced their identity provider, uses a different domain after rebranding or simply ceased to exist.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	return discovery.organizations.Timestamp.IsZero()
}

// SecureLocationList returns a slice of all the available locations.
func (discovery *Discovery) SecureLocationList() []string {
	var locations []string
	for _, currentServer := range discovery.servers.List {
		if currentServer.Type == "secure_internet" {
			locations = append(locations, currentServer.CountryCode)
		}
	}
	return locations
}

// ServerByURL returns the discovery server by the base URL and the according type ("secure_internet", "institute_access")
// An error is returned if and only if nil is returned for the server.
func (discovery *Discovery) ServerByURL(
	baseURL string,
	serverType string,
) (*types.DiscoveryServer, error) {
	for _, currentServer := range discovery.servers.List {
		if currentServer.BaseURL == baseURL && currentServer.Type == serverType {
			return &currentServer, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting server by URL from discovery",
		&GetServerByURLNotFoundError{URL: baseURL, Type: serverType},
	)
}

// ServerByCountryCode returns the discovery server by the country code and the according type ("secure_internet", "institute_access")
// An error is returned if and only if nil is returned for the server.
func (discovery *Discovery) ServerByCountryCode(
	countryCode string,
	serverType string,
) (*types.DiscoveryServer, error) {
	for _, currentServer := range discovery.servers.List {
		if currentServer.CountryCode == countryCode && currentServer.Type == serverType {
			return &currentServer, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting server by country countryCode from discovery",
		&GetServerByCountryCodeNotFoundError{CountryCode: countryCode, Type: serverType},
	)
}

// orgByID returns the discovery organization by the organization ID
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) orgByID(orgID string) (*types.DiscoveryOrganization, error) {
	for _, organization := range discovery.organizations.List {
		if organization.OrgID == orgID {
			return &organization, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting Secure Internet Home URL from discovery",
		&GetOrgByIDNotFoundError{ID: orgID},
	)
}

// SecureHomeArgs returns the secure internet home server arguments:
// - The organization it belongs to
// - The secure internet server itself
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) SecureHomeArgs(
	orgID string,
) (*types.DiscoveryOrganization, *types.DiscoveryServer, error) {
	errorMessage := "failed getting Secure Internet Home arguments from discovery"
	org, orgErr := discovery.orgByID(orgID)

	if orgErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, orgErr)
	}

	// Get a server with the base url
	url := org.SecureInternetHome

	currentServer, serverErr := discovery.ServerByURL(url, "secure_internet")

	if serverErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, serverErr)
	}
	return org, currentServer, nil
}

// DetermineServersUpdate returns whether or not the discovery servers should be updated by contacting the discovery server
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [Implemented] The application MUST always fetch the server_list.json at application start.
// - The application MAY refresh the server_list.json periodically, e.g. once every hour.
func (discovery *Discovery) DetermineServersUpdate() bool {
	// No servers, we should update
	if discovery.servers.Timestamp.IsZero() {
		return true
	}
	// 1 hour from the last update
	shouldUpdateTime := discovery.servers.Timestamp.Add(1 * time.Hour)
	now := time.Now()
	return !now.Before(shouldUpdateTime)
}

// Organizations returns the discovery organizations
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Organizations() (*types.DiscoveryOrganizations, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return &discovery.organizations, nil
	}
	file := "organization_list.json"
	bodyErr := discoFile(file, discovery.organizations.Version, &discovery.organizations)
	if bodyErr != nil {
		// Return previous with an error
		return &discovery.organizations, types.NewWrappedError(
			"failed getting organizations in Discovery",
			bodyErr,
		)
	}
	discovery.organizations.Timestamp = time.Now()
	return &discovery.organizations, nil
}

// Servers returns the discovery servers
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Servers() (*types.DiscoveryServers, error) {
	if !discovery.DetermineServersUpdate() {
		return &discovery.servers, nil
	}
	file := "server_list.json"
	bodyErr := discoFile(file, discovery.servers.Version, &discovery.servers)
	if bodyErr != nil {
		// Return previous with an error
		return &discovery.servers, types.NewWrappedError(
			"failed getting servers in Discovery",
			bodyErr,
		)
	}
	// Update servers timestamp
	discovery.servers.Timestamp = time.Now()
	return &discovery.servers, nil
}

type GetOrgByIDNotFoundError struct {
	ID string
}

func (e GetOrgByIDNotFoundError) Error() string {
	return fmt.Sprintf(
		"No Secure Internet Home found in organizations with ID %s. Please choose your server again",
		e.ID,
	)
}

type GetServerByURLNotFoundError struct {
	URL  string
	Type string
}

func (e GetServerByURLNotFoundError) Error() string {
	return fmt.Sprintf(
		"No institute access server found in organizations with URL %s and type %s. Please choose your server again",
		e.URL,
		e.Type,
	)
}

type GetServerByCountryCodeNotFoundError struct {
	CountryCode string
	Type        string
}

func (e GetServerByCountryCodeNotFoundError) Error() string {
	return fmt.Sprintf(
		"No institute access server found in organizations with country code %s and type %s",
		e.CountryCode,
		e.Type,
	)
}

type GetSecureHomeArgsNotFoundError struct {
	URL string
}

func (e GetSecureHomeArgsNotFoundError) Error() string {
	return fmt.Sprintf(
		"No Secure Internet Home found with URL: %s. Please choose your server again",
		e.URL,
	)
}
