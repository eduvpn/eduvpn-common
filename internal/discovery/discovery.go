// Package discovery implements the server discovery by contacting disco.eduvpn.org and returning the data as a Go structure
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/levenshtein"
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
	// UpdateHeader is the result of the "Last-Modified" header
	UpdateHeader time.Time `json:"go_update_header"`
}

// Organization is a single discovery Organization
type Organization struct {
	// Organization is the embedded public type that is a subset of this thus common Organization
	discotypes.Organization
	// SecureInternetHome is the secure internet home server that belongs to this organization
	// Omitted if none is defined
	SecureInternetHome string `json:"secure_internet_home"`
	// KeywordList is the list of keywords
	// Omitted if none is defined
	KeywordList discotypes.MapOrString `json:"keyword_list,omitempty"`
}

func (o *Organization) Score(search string) int {
	return levenshtein.DiscoveryScore(search, o.DisplayName, o.KeywordList)
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
	// UpdateHeader is the result of the "Last-Modified" header
	UpdateHeader time.Time `json:"go_update_header"`
}

// Server is a single discovery server
type Server struct {
	// Server is the embedded public type that is a subset of this common Server
	discotypes.Server
	// AuthenticationURLTemplate is the template to be used for authentication to skip WAYF
	AuthenticationURLTemplate string `json:"authentication_url_template,omitempty"`
	// KeywordList are the keywords of the server, omitted if empty
	KeywordList discotypes.MapOrString `json:"keyword_list,omitempty"`
	// PublicKeyList are the public keys of the server. Currently not used in this lib but returned by the upstream discovery server
	PublicKeyList []string `json:"public_key_list,omitempty"`
	// SupportContact is the list/slice of support contacts
	SupportContact []string `json:"support_contact,omitempty"`
}

// Matches returns if the search query `str` matches with this server
func (s *Server) Score(search string) int {
	return levenshtein.DiscoveryScore(search, s.DisplayName, s.KeywordList)
}

// Discovery is the main structure used for this package.
type Discovery struct {
	// The httpClient for sending HTTP requests
	httpClient *httpw.Client

	// Organizations represents the organizations that are returned by the discovery server
	OrganizationList Organizations `json:"organizations"`

	// Servers represents the servers that are returned by the discovery server
	ServerList Servers `json:"servers"`
}

// DiscoURL is the URL used for fetching the discovery files and signatures
var DiscoURL = "https://disco.eduvpn.org/v2/"

// file is a helper function that gets a disco JSON and fills the structure with it
// If it was unsuccessful it returns an error.
func (discovery *Discovery) file(ctx context.Context, jsonFile string, previousVersion uint64, last time.Time, structure interface{}) (time.Time, error) {
	var newUpdate time.Time
	// No HTTP client present, create one
	if discovery.httpClient == nil {
		discovery.httpClient = httpw.NewClient(nil)
	}

	// Get json data
	jsonURL, err := httpw.JoinURLPath(DiscoURL, jsonFile)
	if err != nil {
		return newUpdate, err
	}

	var opts *httpw.OptionalParams
	if !last.IsZero() {
		header := http.Header{
			"If-Modified-Since": []string{last.Format(http.TimeFormat)},
		}
		opts = &httpw.OptionalParams{
			Headers: header,
		}
	}
	h, body, err := discovery.httpClient.Do(ctx, "GET", jsonURL, opts)
	if err != nil {
		return newUpdate, err
	}

	lms := h.Get("Last-Modified")
	if lms != "" {
		lm, err := http.ParseTime(lms)
		if err != nil {
			log.Logger.Warningf("failed to parse 'Last-Modified' header: %v", err)
		} else {
			newUpdate = lm
			log.Logger.Debugf("got 'Last-Modified' header: %v", lm)
		}
	} else {
		log.Logger.Warningf("no 'Last-Modified' header found")
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL, err := httpw.JoinURLPath(DiscoURL, sigFile)
	if err != nil {
		return newUpdate, err
	}
	_, sigBody, err := discovery.httpClient.Get(ctx, sigURL)
	if err != nil {
		return newUpdate, err
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
		return newUpdate, err
	}

	// Parse JSON to extract version and list
	if err = json.Unmarshal(body, structure); err != nil {
		return newUpdate, fmt.Errorf("failed parsing discovery file: '%s' from the server with error: %w", jsonFile, err)
	}

	return newUpdate, nil
}

// MarkOrganizationsExpired marks the organizations as expired
func (discovery *Discovery) MarkOrganizationsExpired() {
	// Re-initialize the timestamp to zero
	discovery.OrganizationList.Timestamp = time.Time{}
}

// MarkServersExpired marks the servers as expired
func (discovery *Discovery) MarkServersExpired() {
	// Re-initialize the timestamp to zero
	discovery.ServerList.Timestamp = time.Time{}
}

// DetermineOrganizationsUpdate returns a boolean indicating whether or not the discovery organizations should be updated
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [IMPLEMENTED in client/client.go and here] when the user tries to add new server AND the user did NOT yet choose an organization before; Implemented in Register()
// - [IMPLEMENTED in client/client.go] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
// - [IMPLEMENTED here] NOTE: when the org_id that the user chose previously is no longer available in organization_list.json the application should ask the user to choose their organization (again). This can occur for example when the organization replaced their identity provider, uses a different domain after rebranding or simply ceased to exist.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	if discovery.OrganizationList.Timestamp.IsZero() {
		return true
	}
	if discovery.OrganizationList.UpdateHeader.IsZero() {
		return true
	}
	return false
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
		discovery.MarkOrganizationsExpired()
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
	if discovery.ServerList.UpdateHeader.IsZero() {
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
// The second return value is a boolean that indicates whether a fresh list was updated internally
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Organizations(ctx context.Context) (*Organizations, bool, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return &discovery.OrganizationList, false, nil
	}
	file := "organization_list.json"
	var jsonDecode Organizations
	update, err := discovery.file(ctx, file, discovery.OrganizationList.Version, discovery.OrganizationList.UpdateHeader, &jsonDecode)
	if err != nil {
		statErr := &httpw.StatusError{}
		if errors.As(err, &statErr) {
			if statErr.Status != 304 {
				log.Logger.Warningf("failed to get fresh organizations: %v", err)
			} else {
				discovery.OrganizationList.Timestamp = time.Now()
				log.Logger.Debugf("got 304 for discovery, organization_list.json not modified")
				err = nil
			}
		}
		// Return previous with an error
		orgs, perr := discovery.previousOrganizations()
		if perr != nil {
			log.Logger.Warningf("failed to get previous discovery organizations: %v", perr)
		}
		return orgs, false, err
	}
	if len(jsonDecode.List) == 0 {
		log.Logger.Warningf("fresh organization list is empty")
	} else {
		discovery.OrganizationList = jsonDecode
	}
	discovery.OrganizationList.Timestamp = time.Now()
	if !update.IsZero() {
		discovery.OrganizationList.UpdateHeader = update
	}
	return &discovery.OrganizationList, true, nil
}

// Servers returns the discovery servers
// The second return value is a boolean that indicates whether a fresh list was updated internally
// If there was an error, a cached copy is returned if available.
func (discovery *Discovery) Servers(ctx context.Context) (*Servers, bool, error) {
	if !discovery.DetermineServersUpdate() {
		return &discovery.ServerList, false, nil
	}
	file := "server_list.json"
	var jsonDecode Servers
	update, err := discovery.file(ctx, file, discovery.ServerList.Version, discovery.ServerList.UpdateHeader, &jsonDecode)
	if err != nil {
		statErr := &httpw.StatusError{}
		if errors.As(err, &statErr) {
			if statErr.Status != 304 {
				log.Logger.Warningf("failed to get fresh servers: %v", err)
			} else {
				discovery.ServerList.Timestamp = time.Now()
				log.Logger.Debugf("got 304 for discovery, server_list.json not modified")
				err = nil
			}
		}
		// Return previous with an error
		srvs, perr := discovery.previousServers()
		if perr != nil {
			log.Logger.Warningf("failed to get previous discovery servers: %v", perr)
		}
		return srvs, false, err
	}
	if len(jsonDecode.List) == 0 {
		log.Logger.Warningf("fresh server list is empty")
	} else {
		discovery.ServerList = jsonDecode
	}
	discovery.ServerList.Timestamp = time.Now()
	if !update.IsZero() {
		discovery.ServerList.UpdateHeader = update
	}
	return &discovery.ServerList, true, nil
}

func (discovery *Discovery) UpdateServers(other Discovery) {
	if other.ServerList.Version >= discovery.ServerList.Version {
		discovery.ServerList = other.ServerList
	}
}

func (discovery *Discovery) Copy() (Discovery, error) {
	var dest Discovery
	b, err := json.Marshal(discovery)
	if err != nil {
		return dest, err
	}

	err = json.Unmarshal(b, &dest)
	if err != nil {
		return dest, err
	}

	return dest, nil
}
