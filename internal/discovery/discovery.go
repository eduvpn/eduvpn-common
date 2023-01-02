// Package discovery implements the server discovery by contacting disco.eduvpn.org and returning the data as a Go structure
package discovery

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/verify"
	"github.com/eduvpn/eduvpn-common/types"
	"github.com/go-errors/errors"
)

// Discovery is the main structure used for this package.
type Discovery struct {
	// organizations represents the organizations that are returned by the discovery server
	organizations types.DiscoveryOrganizations

	// servers represents the servers that are returned by the discovery server
	servers types.DiscoveryServers
}

var DiscoURL = "https://disco.eduvpn.org/v2/"

// discoFile is a helper function that gets a disco JSON and fills the structure with it
// If it was unsuccessful it returns an error.
func discoFile(jsonFile string, previousVersion uint64, structure interface{}) error {
	// Get json data
	jsonURL := DiscoURL + jsonFile
	_, body, err := http.Get(jsonURL)
	if err != nil {
		return err
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := DiscoURL + sigFile
	_, sigBody, err := http.Get(sigURL)
	if err != nil {
		return err
	}

	// Verify signature
	// Set this to true when we want to force prehash
	const forcePrehash = false
	ok, err := verify.Verify(
		string(sigBody),
		body,
		jsonFile,
		previousVersion,
		forcePrehash,
	)

	if !ok || err != nil {
		return err
	}

	// Parse JSON to extract version and list
	if err = json.Unmarshal(body, structure); err != nil {
		return errors.WrapPrefix(err,
			fmt.Sprintf("failed getting file: %s from the Discovery server", jsonFile), 0)
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
	var loc []string
	for _, srv := range discovery.servers.List {
		if srv.Type == "secure_internet" {
			loc = append(loc, srv.CountryCode)
		}
	}
	return loc
}

// ServerByURL returns the discovery server by the base URL and the according type ("secure_internet", "institute_access")
// An error is returned if and only if nil is returned for the server.
func (discovery *Discovery) ServerByURL(
	baseURL string,
	srvType string,
) (*types.DiscoveryServer, error) {
	for _, currentServer := range discovery.servers.List {
		if currentServer.BaseURL == baseURL && currentServer.Type == srvType {
			return &currentServer, nil
		}
	}
	return nil, errors.Errorf("no server of type '%s' at URL '%s'", srvType, baseURL)
}

// ServerByCountryCode returns the discovery server by the country code and the according type ("secure_internet", "institute_access")
// An error is returned if and only if nil is returned for the server.
func (discovery *Discovery) ServerByCountryCode(countryCode string, srvType string) (*types.DiscoveryServer, error) {
	for _, srv := range discovery.servers.List {
		if srv.CountryCode == countryCode && srv.Type == srvType {
			return &srv, nil
		}
	}
	return nil, errors.Errorf("no server of type '%s' with country code '%s'", srvType, countryCode)
}

// orgByID returns the discovery organization by the organization ID
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) orgByID(orgID string) (*types.DiscoveryOrganization, error) {
	for _, org := range discovery.organizations.List {
		if org.OrgID == orgID {
			return &org, nil
		}
	}
	return nil, errors.Errorf("no secure internet home found in organization '%s'", orgID)
}

// SecureHomeArgs returns the secure internet home server arguments:
// - The organization it belongs to
// - The secure internet server itself
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) SecureHomeArgs(orgID string) (*types.DiscoveryOrganization, *types.DiscoveryServer, error) {
	org, err := discovery.orgByID(orgID)
	if err != nil {
		return nil, nil, err
	}

	// Get a server with the base url
	srv, err := discovery.ServerByURL(org.SecureInternetHome, "secure_internet")
	if err != nil {
		return nil, nil, err
	}
	return org, srv, nil
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
	upd := discovery.servers.Timestamp.Add(1 * time.Hour)
	return !time.Now().Before(upd)
}

// Organizations returns the discovery organizations
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Organizations() (*types.DiscoveryOrganizations, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return &discovery.organizations, nil
	}
	file := "organization_list.json"
	err := discoFile(file, discovery.organizations.Version, &discovery.organizations)
	if err != nil {
		// Return previous with an error
		return &discovery.organizations, err
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
	err := discoFile(file, discovery.servers.Version, &discovery.servers)
	if err != nil {
		// Return previous with an error
		return &discovery.servers, err
	}
	// Update servers timestamp
	discovery.servers.Timestamp = time.Now()
	return &discovery.servers, nil
}
