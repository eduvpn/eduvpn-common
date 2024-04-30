// Package discovery implements the server discovery by contacting disco.eduvpn.org and returning the data as a Go structure
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/verify"
	discotypes "github.com/eduvpn/eduvpn-common/types/discovery"
)

// HasCache denotes whether or not we have an embedded cache available
var HasCache bool

// Organizations are the list of organizations from https://disco.eduvpn.org/v2/organization_list.json
type Organizations struct {
	// Version is the version field in discovery. The Go library checks this for rollbacks
	Version uint64 `json:"v"`
	// List is the list of organizations, omitted if empty
	List []Organization `json:"organization_list,omitempty"`
	// Timestamp is the timestamp that is internally used by the Go library to keep track
	// of when the organizations were last updated
	Timestamp time.Time `json:"go_timestamp"`
}

// Organization is a single discovery Organization
type Organization struct {
	// Organization is the embedded public type that is a subset of this thus common Organization
	discotypes.Organization
	// SecureInternetHome is the secure internet home server that belongs to this organization
	// Omitted if none is defined
	SecureInternetHome string `json:"secure_internet_home"`
}

// Servers are the list of servers from https://disco.eduvpn.org/v2/server_list.json
type Servers struct {
	// Version is the version field in discovery. The Go library checks this for rollbacks
	Version uint64 `json:"v"`
	// List is the list of servers, omitted if empty
	List []Server `json:"server_list,omitempty"`
	// Timestamp is a timestamp that is internally used by the Go library to keek track
	// of when the servers were last updated
	Timestamp time.Time `json:"go_timestamp"`
}

// Server is a single discovery server
type Server struct {
	// Server is the embedded public type that is a subset of this common Server
	discotypes.Server
	// AuthenticationURLTemplate is the template to be used for authentication to skip WAYF
	AuthenticationURLTemplate string `json:"authentication_url_template,omitempty"`
	// CountryCode is the country code for the server in case of secure internet, e.g. NL
	CountryCode string `json:"country_code,omitempty"`
	// PublicKeyList are the public keys of the server. Currently not used in this lib but returned by the upstream discovery server
	PublicKeyList []string `json:"public_key_list,omitempty"`
	// SupportContact is the list/slice of support contacts
	SupportContact []string `json:"support_contact,omitempty"`
}

// Discovery is the main structure used for this package.
type Discovery struct {
	// The httpClient for sending HTTP requests
	httpClient *http.Client

	// Organizations represents the organizations that are returned by the discovery server
	OrganizationList Organizations `json:"organizations"`

	// Servers represents the servers that are returned by the discovery server
	ServerList Servers `json:"servers"`
}

// DiscoURL is the URL used for fetching the discovery files and signatures
var DiscoURL = "https://disco.eduvpn.org/v2/"

// file is a helper function that gets a disco JSON and fills the structure with it
// If it was unsuccessful it returns an error.
func (discovery *Discovery) file(ctx context.Context, jsonFile string, previousVersion uint64, structure interface{}) error {
	// No HTTP client present, create one
	if discovery.httpClient == nil {
		discovery.httpClient = http.NewClient(nil)
	}

	// Get json data
	jsonURL, err := http.JoinURLPath(DiscoURL, jsonFile)
	if err != nil {
		return err
	}
	_, body, err := discovery.httpClient.Get(ctx, jsonURL)
	if err != nil {
		return err
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL, err := http.JoinURLPath(DiscoURL, sigFile)
	if err != nil {
		return err
	}
	_, sigBody, err := discovery.httpClient.Get(ctx, sigURL)
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
		return fmt.Errorf("failed parsing discovery file: '%s' from the server with error: %w", jsonFile, err)
	}

	return nil
}

// MarkOrganizationsExpired marks the organizations as expired
func (discovery *Discovery) MarkOrganizationsExpired() {
	// Re-initialize the timestamp to zero
	discovery.OrganizationList.Timestamp = time.Time{}
}

// DetermineOrganizationsUpdate returns a boolean indicating whether or not the discovery organizations should be updated
// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [IMPLEMENTED in client/server.go] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [IMPLEMENTED in client/server.go] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
// - [IMPLEMENTED using a custom error message, and in client/server.go] NOTE: when the org_id that the user chose previously is no longer available in organization_list.json the application should ask the user to choose their organization (again). This can occur for example when the organization replaced their identity provider, uses a different domain after rebranding or simply ceased to exist.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	return discovery.OrganizationList.Timestamp.IsZero()
}

// SecureLocationList returns a slice of all the available locations.
func (discovery *Discovery) SecureLocationList() []string {
	var loc []string
	for _, srv := range discovery.ServerList.List {
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
) (*Server, error) {
	for _, currentServer := range discovery.ServerList.List {
		if currentServer.BaseURL == baseURL && currentServer.Type == srvType {
			return &currentServer, nil
		}
	}
	return nil, fmt.Errorf("no server of type '%s' at URL '%s'", srvType, baseURL)
}

// ErrCountryNotFound is used when the secure internet country cannot be found
type ErrCountryNotFound struct {
	CountryCode string
}

func (cnf *ErrCountryNotFound) Error() string {
	return fmt.Sprintf("no secure internet server with country code: '%s'", cnf.CountryCode)
}

// ServerByCountryCode returns the discovery server by the country code
// An error is returned if and only if nil is returned for the server.
func (discovery *Discovery) ServerByCountryCode(countryCode string) (*Server, error) {
	for _, srv := range discovery.ServerList.List {
		if srv.CountryCode == countryCode && srv.Type == "secure_internet" {
			return &srv, nil
		}
	}
	return nil, &ErrCountryNotFound{CountryCode: countryCode}
}

// orgByID returns the discovery organization by the organization ID
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) orgByID(orgID string) (*Organization, error) {
	for _, org := range discovery.OrganizationList.List {
		if org.OrgID == orgID {
			return &org, nil
		}
	}
	return nil, fmt.Errorf("no secure internet home found in organization '%s'", orgID)
}

// SecureHomeArgs returns the secure internet home server arguments:
// - The organization it belongs to
// - The secure internet server itself
// An error is returned if and only if nil is returned for the organization.
func (discovery *Discovery) SecureHomeArgs(orgID string) (*Organization, *Server, error) {
	org, err := discovery.orgByID(orgID)
	if err != nil {
		discovery.MarkOrganizationsExpired()
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
	if discovery.ServerList.Timestamp.IsZero() {
		return true
	}
	// 1 hour from the last update
	upd := discovery.ServerList.Timestamp.Add(1 * time.Hour)
	return !time.Now().Before(upd)
}

func (discovery *Discovery) previousOrganizations() (*Organizations, error) {
	// If the version field is not zero then we have a cached struct
	// We also immediately return this copy if we have no embedded JSON
	if discovery.OrganizationList.Version != 0 || !HasCache {
		return &discovery.OrganizationList, nil
	}

	// We do not have a cached struct, this we need to get it using the embedded JSON
	var eo Organizations
	if err := json.Unmarshal(eOrganizations, &eo); err != nil {
		return nil, fmt.Errorf("failed parsing discovery organizations from the embedded cache with error: %w", err)
	}
	discovery.OrganizationList = eo
	return &eo, nil
}

func (discovery *Discovery) previousServers() (*Servers, error) {
	// If the version field is not zero then we have a cached struct
	// We also immediately return this copy if we have no embedded JSON
	if discovery.ServerList.Version != 0 || !HasCache {
		return &discovery.ServerList, nil
	}

	// We do not have a cached struct, this we need to get it using the embedded JSON
	var es Servers
	if err := json.Unmarshal(eServers, &es); err != nil {
		return nil, fmt.Errorf("failed parsing discovery servers from the embedded cache with error: %w", err)
	}
	discovery.ServerList = es
	return &es, nil
}

// Organizations returns the discovery organizations
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Organizations(ctx context.Context) (*Organizations, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return &discovery.OrganizationList, nil
	}
	file := "organization_list.json"
	err := discovery.file(ctx, file, discovery.OrganizationList.Version, &discovery.OrganizationList)
	if err != nil {
		// Return previous with an error
		orgs, perr := discovery.previousOrganizations()
		if perr != nil {
			log.Logger.Warningf("failed to get previous discovery organizations: %v", perr)
		}
		return orgs, err
	}
	discovery.OrganizationList.Timestamp = time.Now()
	return &discovery.OrganizationList, nil
}

// Servers returns the discovery servers
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Servers(ctx context.Context) (*Servers, error) {
	if !discovery.DetermineServersUpdate() {
		return &discovery.ServerList, nil
	}
	file := "server_list.json"
	err := discovery.file(ctx, file, discovery.ServerList.Version, &discovery.ServerList)
	if err != nil {
		// Return previous with an error
		// TODO: Log here if we fail to get previous
		srvs, perr := discovery.previousServers()
		if perr != nil {
			log.Logger.Warningf("failed to get previous discovery servers: %v", perr)
		}
		return srvs, err
	}
	// Update servers timestamp
	discovery.ServerList.Timestamp = time.Now()
	return &discovery.ServerList, nil
}
