// Package test implements utilities for testing
package test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/go-errors/errors"
)

type TestServer struct {
	*httptest.Server
}

func NewServer(handler http.Handler) *TestServer {
	s := httptest.NewTLSServer(handler)

	return &TestServer{s}
}

// Client returns a test client that trusts the HTTPS certificates
func (srv *TestServer) Client() (*httpw.Client, error) {
	// Get the certs from the test server
	certs := x509.NewCertPool()
	for _, c := range srv.TLS.Certificates {
		roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
		if err != nil {
			return nil, errors.WrapPrefix(err, "failed to parse root certificate", 0)
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
