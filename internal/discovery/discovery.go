package discovery

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/verify"
	"github.com/eduvpn/eduvpn-common/types"
)

type Discovery struct {
	Organizations types.DiscoveryOrganizations
	Servers       types.DiscoveryServers
}

// Helper function that gets a disco json and fills the structure with it
func getDiscoFile(jsonFile string, previousVersion uint64, structure interface{}) error {
	errorMessage := fmt.Sprintf("failed getting file: %s from the Discovery server", jsonFile)
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	_, fileBody, fileErr := http.HTTPGet(fileURL)

	if fileErr != nil {
		return types.NewWrappedError(errorMessage, fileErr)
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	_, sigBody, sigFileErr := http.HTTPGet(sigURL)

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

// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [TODO] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [TODO] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
// - [IMPLEMENTED using a custom error message] NOTE: when the org_id that the user chose previously is no longer available in organization_list.json the application should ask the user to choose their organization (again). This can occur for example when the organization replaced their identity provider, uses a different domain after rebranding or simply ceased to exist.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	return discovery.Organizations.Timestamp.IsZero()
}

func (discovery *Discovery) GetSecureLocationList() []string {
	var locations []string
	for _, currentServer := range discovery.Servers.List {
		if currentServer.Type == "secure_internet" {
			locations = append(locations, currentServer.CountryCode)
		}
	}
	return locations
}

func (discovery *Discovery) GetServerByURL(
	url string,
	_type string,
) (*types.DiscoveryServer, error) {
	for _, currentServer := range discovery.Servers.List {
		if currentServer.BaseURL == url && currentServer.Type == _type {
			return &currentServer, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting server by URL from discovery",
		&GetServerByURLNotFoundError{URL: url, Type: _type},
	)
}

func (discovery *Discovery) GetServerByCountryCode(
	code string,
	_type string,
) (*types.DiscoveryServer, error) {
	for _, currentServer := range discovery.Servers.List {
		if currentServer.CountryCode == code && currentServer.Type == _type {
			return &currentServer, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting server by country code from discovery",
		&GetServerByCountryCodeNotFoundError{CountryCode: code, Type: _type},
	)
}

func (discovery *Discovery) getOrgByID(orgID string) (*types.DiscoveryOrganization, error) {
	for _, organization := range discovery.Organizations.List {
		if organization.OrgID == orgID {
			return &organization, nil
		}
	}
	return nil, types.NewWrappedError(
		"failed getting Secure Internet Home URL from discovery",
		&GetOrgByIDNotFoundError{ID: orgID},
	)
}

func (discovery *Discovery) GetSecureHomeArgs(
	orgID string,
) (*types.DiscoveryOrganization, *types.DiscoveryServer, error) {
	errorMessage := "failed getting Secure Internet Home arguments from discovery"
	org, orgErr := discovery.getOrgByID(orgID)

	if orgErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, orgErr)
	}

	// Get a server with the base url
	url := org.SecureInternetHome

	currentServer, serverErr := discovery.GetServerByURL(url, "secure_internet")

	if serverErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, serverErr)
	}
	return org, currentServer, nil
}

// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [Implemented] The application MUST always fetch the server_list.json at application start.
// - The application MAY refresh the server_list.json periodically, e.g. once every hour.
func (discovery *Discovery) DetermineServersUpdate() bool {
	// No servers, we should update
	if discovery.Servers.Timestamp.IsZero() {
		return true
	}
	// 1 hour from the last update
	shouldUpdateTime := discovery.Servers.Timestamp.Add(1 * time.Hour)
	now := time.Now()
	return !now.Before(shouldUpdateTime)
}

// Get the organization list
func (discovery *Discovery) GetOrganizationsList() (*types.DiscoveryOrganizations, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return &discovery.Organizations, nil
	}
	file := "organization_list.json"
	bodyErr := getDiscoFile(file, discovery.Organizations.Version, &discovery.Organizations)
	if bodyErr != nil {
		// Return previous with an error
		return &discovery.Organizations, types.NewWrappedError(
			"failed getting organizations in Discovery",
			bodyErr,
		)
	}
	discovery.Organizations.Timestamp = time.Now()
	return &discovery.Organizations, nil
}

// Get the server list
func (discovery *Discovery) GetServersList() (*types.DiscoveryServers, error) {
	if !discovery.DetermineServersUpdate() {
		return &discovery.Servers, nil
	}
	file := "server_list.json"
	bodyErr := getDiscoFile(file, discovery.Servers.Version, &discovery.Servers)
	if bodyErr != nil {
		// Return previous with an error
		return &discovery.Servers, types.NewWrappedError(
			"failed getting servers in Discovery",
			bodyErr,
		)
	}
	// Update servers timestamp
	discovery.Servers.Timestamp = time.Now()
	return &discovery.Servers, nil
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
