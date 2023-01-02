package discovery

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// setupFileServer sets up a file server with a directory
func setupFileServer(t *testing.T, directory string) (*http.Server) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to setup discovery file server")
	}
	s := &http.Server{Handler: http.FileServer(http.Dir(directory))}
	go s.Serve(listener)

	// Override the global disco URL with the local file server
	port := listener.Addr().(*net.TCPAddr).Port
	DiscoURL = fmt.Sprintf("http://127.0.0.1:%d/", port)

	return s
}

// TestServers tests whether or not we can obtain discovery servers
// It setups up a file server using the 'test_files' directory
func TestServers(t *testing.T) {
	s := setupFileServer(t, "test_files")
	d := &Discovery{}
	// get servers
	s1, err := d.Servers()
	if err != nil {
		t.Fatalf("Failed getting servers: %v", err)
	}

	// Shutdown the server
	err = s.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	// Test if we get the same cached copy
	s2, err := d.Servers()
	// We should not get an error as the timestamp is not expired
	if err != nil {
		t.Fatalf("Got a servers error after shutting down server: %v", err)
	}
	if s1 != s2 {
		t.Fatalf("Servers copies not equal after shutting down file server")
	}

	// Force expired, 1 hour in the past
	d.servers.Timestamp = time.Now().Add(-1 * time.Hour)

	s3, err := d.Servers()
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
	s := setupFileServer(t, "test_files")
	d := &Discovery{}
	// get servers
	s1, err := d.Organizations()
	if err != nil {
		t.Fatalf("Failed getting organizations: %v", err)
	}

	// Shutdown the server
	err = s.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown server: %v", err)
	}

	// Test if we get the same cached copy
	// We should not get an error as the timestamp is not zero
	s2, err := d.Organizations()
	if err != nil {
		t.Fatalf("Got an organizations error after shutting down file server: %v", err)
	}
	if s1 != s2 {
		t.Fatalf("Organizations copies not equal after shutting down file server")
	}
}
