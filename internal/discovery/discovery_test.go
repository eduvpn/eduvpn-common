package discovery

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/test"
	discotypes "github.com/eduvpn/eduvpn-common/types/discovery"
)

// TestServers tests whether or not we can obtain discovery servers
// It setups up a file server using the 'test_files' directory
func TestServers(t *testing.T) {
	handler := http.FileServer(http.Dir("test_files"))
	s := test.NewServer(handler, nil)
	DiscoURL = s.URL
	c, err := s.Client()
	if err != nil {
		t.Fatalf("Failed to get HTTP test client: %v", err)
	}
	d := &Discovery{httpClient: c}
	// get servers
	_, fresh, err := d.Servers(context.Background())
	if !fresh {
		t.Fatalf("Did not obtain the server list fresh")
	}
	if err != nil {
		t.Fatalf("Failed getting servers: %v", err)
	}

	// force expired
	d.ServerList.Timestamp = time.Now().Add(-1 * time.Hour)

	// override the last server with a keyword list
	// and make sure a new fetch doesn't copy over the keyword list
	// to the last entry
	d.ServerList.List[len(d.ServerList.List)-1] = Server{
		Server: discotypes.Server{
			BaseURL: "https://example.org/",
			DisplayName: map[string]string{
				"en": "example",
			},
			Type: "institute_access",
		},
		KeywordList: map[string]string{
			"en": "test bla",
		},
		SupportContact: []string{"mailto:test@example.org"},
	}
	// conditional requests: this should not be fetched fresh
	_, fresh, err = d.Servers(context.Background())
	if fresh {
		t.Fatalf("Obtained the server list fresh with conditional requests")
	}
	if err != nil {
		t.Fatalf("Failed getting servers after inserting a mock entry: %v", err)
	}
	// mock conditional requests
	d.ServerList.UpdateHeader = time.Time{}
	s1, fresh, err := d.Servers(context.Background())
	if !fresh {
		t.Fatalf("Did not obtain the server list fresh with mocked conditional request and mocked entry")
	}
	if err != nil {
		t.Fatalf("Failed getting servers after inserting a mock entry and mocking conditional request: %v", err)
	}
	if kws := s1.List[len(s1.List)-1].KeywordList; kws != nil {
		t.Fatalf("KeywordList is not nil when getting a fresh server list after inserting a mock entry: %v", kws)
	}

	// Shutdown the server
	s.Close()
	// Test if we get the same cached copy
	s2, fresh, err := d.Servers(context.Background())
	// We should not get an error as the timestamp is not expired
	if fresh {
		t.Fatalf("The server list was obtained fresh")
	}
	if err != nil {
		t.Fatalf("Got a servers error after shutting down server: %v", err)
	}
	if s1 != s2 {
		t.Fatalf("Servers copies not equal after shutting down file server")
	}

	// Force expired, 1 hour in the past
	// we should return the previous with an error
	d.ServerList.Timestamp = time.Now().Add(-1 * time.Hour)

	s3, fresh, err := d.Servers(context.Background())
	if fresh {
		t.Fatalf("Server list was gotten fresh")
	}
	// Now we expect an error with the cached copy
	if err == nil {
		t.Fatalf("Got a servers nil error after shutting down file server and expired")
	}
	if s1 != s3 {
		t.Fatalf("Servers copies not equal after shutting down file server and expired")
	}
}

// TestOrganizations tests whether or not we can obtain discovery organizations
// It setups up a file server using the 'test_files' directory
func TestOrganizations(t *testing.T) {
	handler := http.FileServer(http.Dir("test_files"))
	s := test.NewServer(handler, nil)
	DiscoURL = s.URL
	c, err := s.Client()
	if err != nil {
		t.Fatalf("Failed to get HTTP test client: %v", err)
	}
	d := &Discovery{httpClient: c}
	// get servers
	_, fresh, err := d.Organizations(context.Background())
	if !fresh {
		t.Fatalf("The organization list was not obtained fresh")
	}
	if err != nil {
		t.Fatalf("Failed getting organizations: %v", err)
	}
	if !fresh {
		t.Fatalf("Did not get a fresh organization list")
	}

	// force expired
	d.OrganizationList.Timestamp = time.Now().Add(-4 * time.Hour)

	// override the last organization with a keyword list
	// and make sure a new fetch doesn't copy over the keyword list
	// to the last entry
	d.OrganizationList.List[len(d.OrganizationList.List)-1] = Organization{
		Organization: discotypes.Organization{
			DisplayName: map[string]string{
				"en": "example",
			},
			OrgID: "example_orgid",
		},
		SecureInternetHome: "example.org",
		KeywordList: map[string]string{
			"en": "test bla",
		},
	}

	_, fresh, err = d.Organizations(context.Background())
	if fresh {
		t.Fatalf("Obtained the organization list fresh with conditional requests")
	}
	if err != nil {
		t.Fatalf("Failed getting organizations after inserting a mock entry: %v", err)
	}
	// mock conditional requests
	d.OrganizationList.UpdateHeader = time.Time{}
	s1, fresh, err := d.Organizations(context.Background())
	if !fresh {
		t.Fatalf("Did not obtain the organization list fresh after inserting a mock entry, faking expiry and mocking conditional request")
	}
	if err != nil {
		t.Fatalf("Failed getting organizations after inserting a mock entry and faking conditional request: %v", err)
	}
	if kws := s1.List[len(s1.List)-1].KeywordList; kws != nil {
		t.Fatalf("KeywordList is not nil when getting a fresh organization list after inserting a mock entry: %v", kws)
	}

	// Shutdown the server
	s.Close()
	// Test if we get the same cached copy
	// We should not get an error as the timestamp is not zero
	s2, fresh, err := d.Organizations(context.Background())
	if fresh {
		t.Fatalf("The organization list is freshly obtained")
	}
	if err != nil {
		t.Fatalf("Got an organizations error after shutting down file server: %v", err)
	}
	if s1 != s2 {
		t.Fatalf("Organizations copies not equal after shutting down file server")
	}
}

// TestSecureLocationList tests the function for getting a list of secure internet servers
func TestSecureLocationList(t *testing.T) {
	d := Discovery{
		ServerList: Servers{
			Version: 1,
			List: []Server{
				// institute access server, this should not be found
				{Server: discotypes.Server{Type: "institute_access"}},
				// secure internet servers, these should be found
				{Server: discotypes.Server{Type: "secure_internet", CountryCode: "b"}},
				{Server: discotypes.Server{Type: "secure_internet", CountryCode: "c"}},
				// Unexpected type, this should not be found
				{Server: discotypes.Server{Type: "test", CountryCode: "d"}},
			},
		},
	}

	cc := d.SecureLocationList()
	want := []string{"b", "c"}

	if !reflect.DeepEqual(cc, want) {
		t.Fatalf("Secure location list is not equal. Got: %v, Want: %v", cc, want)
	}
}

// TestServerByURL tests the function for getting a server by the Base URL and type
func TestServerByURL(t *testing.T) {
	d := Discovery{
		ServerList: Servers{
			Version: 1,
			List: []Server{
				// institute access server
				{Server: discotypes.Server{BaseURL: "a", Type: "institute_access"}},
				// secure internet servers
				{Server: discotypes.Server{BaseURL: "b", Type: "secure_internet"}},
				// Unexpected type, this should not be found
				{Server: discotypes.Server{BaseURL: "d", Type: "test"}},
			},
		},
	}
	// Institute Access: Can be found
	_, err := d.ServerByURL("a", "institute_access")
	if err != nil {
		t.Fatalf("Got error: %v, when getting a server by url with parameters 'a' and 'institute_access'", err)
	}

	// Institute Access: Cannot be found
	_, err = d.ServerByURL("b", "institute_access")
	if err == nil {
		t.Fatal("Got no error, when getting a non-existing server by url with parameters 'b' and 'institute_access'")
	}

	// Secure Internet: Can be found
	_, err = d.ServerByURL("b", "secure_internet")
	if err != nil {
		t.Fatalf("Got error: %v, when getting a server by url with parameters 'b' and 'secure_internet'", err)
	}

	// Secure Internet: Cannot be found because of invalid type
	_, err = d.ServerByURL("d", "secure_internet")
	if err == nil {
		t.Fatal("Got no error, when getting a non-existing server by url with parameters 'd' and 'secure_internet'")
	}
}

// TestServerByCountryCode tests the function for getting a server by the country code
func TestServerByCountryCode(t *testing.T) {
	s1 := Server{Server: discotypes.Server{Type: "secure_internet", CountryCode: "a"}}
	d := Discovery{
		ServerList: Servers{
			Version: 1,
			List: []Server{
				// secure internet server
				s1,
				// Unexpected types, these should not be found
				{Server: discotypes.Server{Type: "institute_access", CountryCode: "b"}},
				{Server: discotypes.Server{Type: "test", CountryCode: "c"}},
			},
		},
	}
	// Institute Access: Can be found
	s, err := d.ServerByCountryCode("a")
	if err != nil {
		t.Fatalf("Got error: %v, when getting a server by country code 'a'", err)
	}
	if s.CountryCode != s1.CountryCode || s.Type != s1.Type {
		t.Fatalf("Server with country code 'a' not equal, Got: %v, Want: %v", s, s1)
	}

	// Others: Cannot be found
	_, err = d.ServerByCountryCode("b")
	if err == nil {
		t.Fatal("Got no error when getting a server by country code 'b'")
	}
	_, err = d.ServerByCountryCode("c")
	if err == nil {
		t.Fatal("Got no error when getting a server by country code 'c'")
	}
}

// TestOrgByID tests the function for getting an organization by ID
func TestOrgByID(t *testing.T) {
	o1 := discotypes.Organization{OrgID: "a"}
	d := Discovery{
		OrganizationList: Organizations{
			Version: 1,
			List: []Organization{
				{Organization: o1},
			},
		},
	}
	o, err := d.orgByID("a")
	if err != nil {
		t.Fatal("Got an error when getting an organization with ID: 'a'")
	}
	if o.OrgID != o1.OrgID {
		t.Fatalf("Organizations not equal, Got: %v, Want: %v", o, o1)
	}
	_, err = d.orgByID("b")
	if err == nil {
		t.Fatal("Got no error when searching for non-existing organization 'b'")
	}
}

// TestSecureHomeArgs tests the function for getting an organization and matching secure internet server by organization ID
func TestSecureHomeArgs(t *testing.T) {
	o1 := Organization{Organization: discotypes.Organization{OrgID: "id"}, SecureInternetHome: "a"}
	s1 := discotypes.Server{BaseURL: "a", Type: "secure_internet"}
	d := Discovery{
		OrganizationList: Organizations{
			Version: 1,
			List: []Organization{
				{Organization: discotypes.Organization{OrgID: "id2"}, SecureInternetHome: "c"},
				o1,
			},
		},
		ServerList: Servers{
			Version: 1,
			List: []Server{
				{Server: s1},
				{Server: discotypes.Server{BaseURL: "b"}},
			},
		},
	}

	// Args found
	o, s, err := d.SecureHomeArgs("id")
	if err != nil {
		t.Fatalf("Got error: %v, when getting secure home arguments with ID: 'id'", err)
	}
	if o.OrgID != o1.OrgID || o.SecureInternetHome != o1.SecureInternetHome {
		t.Fatalf("Organizations not equal for secure home arguments, Got: %v, Want: %v", o, o1)
	}
	if s.BaseURL != s1.BaseURL {
		t.Fatalf("Servers not equal for secure home arguments, Got: %v, Want: %v", s, s1)
	}
	// Args not found because no matching secure internet server
	_, _, err = d.SecureHomeArgs("id2")
	if err == nil {
		t.Fatal("Got no error, when getting non-matching secure home arguments with ID: 'id2'")
	}

	// Args not found because no organization
	_, _, err = d.SecureHomeArgs("id3")
	if err == nil {
		t.Fatal("Got no error, when getting non-existing secure home arguments with ID: 'id3'")
	}
}
