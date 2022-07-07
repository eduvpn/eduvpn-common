package discovery

import (
	"encoding/json"
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/http"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
	"github.com/jwijenbergh/eduvpn-common/internal/verify"
)

type Discovery struct {
	Organizations types.DiscoveryOrganizations
	Servers       types.DiscoveryServers
	FSM           *fsm.FSM
	Logger        *log.FileLogger
}

// Helper function that gets a disco json
func getDiscoFile(jsonFile string, previousVersion uint64, structure interface{}) (string, error) {
	errorMessage := fmt.Sprintf("failed getting file: %s from the Discovery server", jsonFile)
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	_, fileBody, fileErr := http.HTTPGet(fileURL)

	if fileErr != nil {
		return "", &types.WrappedErrorMessage{Message: errorMessage, Err: fileErr}
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	_, sigBody, sigFileErr := http.HTTPGet(sigURL)

	if sigFileErr != nil {
		return "", &types.WrappedErrorMessage{Message: errorMessage, Err: sigFileErr}
	}

	// Verify signature
	// Set this to true when we want to force prehash
	forcePrehash := false
	verifySuccess, verifyErr := verify.Verify(string(sigBody), fileBody, jsonFile, previousVersion, forcePrehash)

	if !verifySuccess || verifyErr != nil {
		return "", &types.WrappedErrorMessage{Message: errorMessage, Err: verifyErr}
	}

	// Parse JSON to extract version and list
	jsonErr := json.Unmarshal(fileBody, structure)

	if jsonErr != nil {
		return "", &types.WrappedErrorMessage{Message: errorMessage, Err: jsonErr}
	}

	return string(fileBody), nil
}

func (discovery *Discovery) Init(fsm *fsm.FSM, logger *log.FileLogger) {
	discovery.FSM = fsm
	discovery.Logger = logger
}

// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [TODO] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [TODO] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	return discovery.Organizations.Timestamp == 0
}

func (discovery *Discovery) GetSecureLocationList() []string {
	var locations []string
	for _, server := range discovery.Servers.List {
		if server.Type == "secure_internet" {
			locations = append(locations, server.CountryCode)
		}
	}
	return locations
}

func (discovery *Discovery) GetServerByURL(url string, _type string) (*types.DiscoveryServer, error) {
	for _, server := range discovery.Servers.List {
		if server.BaseURL == url && server.Type == _type {
			return &server, nil
		}
	}
	return nil, &types.WrappedErrorMessage{Message: "failed getting server by URL from discovery", Err: &GetServerByURLNotFoundError{URL: url, Type: _type}}
}

func (discovery *Discovery) GetServerByCountryCode(code string, _type string) (*types.DiscoveryServer, error) {
	for _, server := range discovery.Servers.List {
		if server.CountryCode == code && server.Type == _type {
			return &server, nil
		}
	}
	return nil, &types.WrappedErrorMessage{Message: "failed getting server by country code from discovery", Err: &GetServerByCountryCodeNotFoundError{CountryCode: code, Type: _type}}
}

func (discovery *Discovery) getOrgByID(orgID string) (*types.DiscoveryOrganization, error) {
	for _, organization := range discovery.Organizations.List {
		if organization.OrgId == orgID {
			return &organization, nil
		}
	}
	return nil, &types.WrappedErrorMessage{Message: "failed getting Secure Internet Home URL from discovery", Err: &GetOrgByIDNotFoundError{ID: orgID}}
}

func (discovery *Discovery) GetSecureHomeArgs(orgID string) (*types.DiscoveryOrganization, *types.DiscoveryServer, error) {
	errorMessage := "failed getting Secure Internet Home arguments from discovery"
	org, orgErr := discovery.getOrgByID(orgID)

	if orgErr != nil {
		return nil, nil, &types.WrappedErrorMessage{Message: errorMessage, Err: orgErr}
	}

	// Get a server with the base url
	url := org.SecureInternetHome

	server, serverErr := discovery.GetServerByURL(url, "secure_internet")

	if serverErr != nil {
		return nil, nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}
	return org, server, nil
}

// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [Implemented] The application MUST always fetch the server_list.json at application start.
// - The application MAY refresh the server_list.json periodically, e.g. once every hour.
func (discovery *Discovery) DetermineServersUpdate() bool {
	// No servers, we should update
	if discovery.Servers.Timestamp == 0 {
		return true
	}
	// 1 hour from the last update
	should_update_time := discovery.Servers.Timestamp + 3600
	now := util.GenerateTimeSeconds()
	if now >= should_update_time {
		return true
	}
	discovery.Logger.Log(log.LOG_INFO, "No update needed for servers, 1h is not passed yet")
	return false
}

// Get the organization list
func (discovery *Discovery) GetOrganizationsList() (string, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return discovery.Organizations.RawString, nil
	}
	file := "organization_list.json"
	body, bodyErr := getDiscoFile(file, discovery.Organizations.Version, &discovery.Organizations)
	if bodyErr != nil {
		// Return previous with an error
		return discovery.Organizations.RawString, &types.WrappedErrorMessage{Message: "failed getting organizations in Discovery", Err: bodyErr}
	}
	discovery.Organizations.RawString = body
	discovery.Organizations.Timestamp = util.GenerateTimeSeconds()
	return discovery.Organizations.RawString, nil
}

// Get the server list
func (discovery *Discovery) GetServersList() (string, error) {
	if !discovery.DetermineServersUpdate() {
		return discovery.Servers.RawString, nil
	}
	file := "server_list.json"
	body, bodyErr := getDiscoFile(file, discovery.Servers.Version, &discovery.Servers)
	if bodyErr != nil {
		// Return previous with an error
		return discovery.Servers.RawString, &types.WrappedErrorMessage{Message: "failed getting servers in Discovery", Err: bodyErr}
	}
	// Update servers timestamp
	discovery.Servers.RawString = body
	discovery.Servers.Timestamp = util.GenerateTimeSeconds()
	return discovery.Servers.RawString, nil
}

type GetOrgByIDNotFoundError struct {
	ID string
}

func (e GetOrgByIDNotFoundError) Error() string {
	return fmt.Sprintf("No Secure Internet Home found in organizations with ID %s", e.ID)
}

type GetServerByURLNotFoundError struct {
	URL  string
	Type string
}

func (e GetServerByURLNotFoundError) Error() string {
	return fmt.Sprintf("No institute access server found in organizations with URL %s and type %s", e.URL, e.Type)
}

type GetServerByCountryCodeNotFoundError struct {
	CountryCode string
	Type        string
}

func (e GetServerByCountryCodeNotFoundError) Error() string {
	return fmt.Sprintf("No institute access server found in organizations with country code %s and type %s", e.CountryCode, e.Type)
}

type GetSecureHomeArgsNotFoundError struct {
	URL string
}

func (e GetSecureHomeArgsNotFoundError) Error() string {
	return fmt.Sprintf("No Secure Internet Home found with URL: %s", e.URL)
}
