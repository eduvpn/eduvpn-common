// Package test implements utilities for testing
package test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
)

type Server struct {
	*httptest.Server
}

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
	// Override the client such that it only trusts the test server cert
	client := httpw.NewClient()
	client.Client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certs,
		},
	}
	return client, nil
}
