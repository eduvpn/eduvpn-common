// Package test implements utilities for testing
package test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
)

// Server wraps a HTTP test server
type Server struct {
	*httptest.Server
}

// NewServer creates a new test server
func NewServer(handler http.Handler) *Server {
	s := httptest.NewTLSServer(handler)
	return &Server{s}
}

// Client returns a test client that trusts the HTTPS certificates
func (srv *Server) Client() (*httpw.Client, error) {
	// Get the certs from the test server
	certs := x509.NewCertPool()
	for _, c := range srv.TLS.Certificates {
		roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			return nil, err
		}
		for _, root := range roots {
			certs.AddCert(root)
		}
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certs,
			},
		},
	}
	// Override the client such that it only trusts the test server cert
	httpC := httpw.NewClient(client)
	return httpC, nil
}
