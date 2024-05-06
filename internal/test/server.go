// Package test implements utilities for testing
package test

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
)

// Server wraps a HTTP test server
type Server struct {
	*httptest.Server
}

// NewServer creates a new test server
func NewServer(handler http.Handler, listener net.Listener) *Server {
	if listener == nil {
		s := httptest.NewTLSServer(handler)
		return &Server{s}
	}

	s := httptest.NewUnstartedServer(handler)
	s.Listener.Close()
	s.Listener = listener
	s.StartTLS()
	return &Server{s}
}

type HandlerPath struct {
	Method          string
	Path            string
	Response        string
	ResponseHandler func(http.ResponseWriter, *http.Request)
	ResponseCode    int
}

func (hp *HandlerPath) HandlerFunc() func(http.ResponseWriter, *http.Request) {
	if hp.ResponseHandler != nil {
		return hp.ResponseHandler
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != hp.Method {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(hp.ResponseCode)
		w.Write([]byte(hp.Response))
	}
}

// NewServerWithHandles creates a new test servers with path and responses
func NewServerWithHandles(hps []HandlerPath, listener net.Listener) *Server {
	mux := http.NewServeMux()
	for _, hp := range hps {
		hp := hp
		mux.HandleFunc(hp.Path, hp.HandlerFunc())
	}
	return NewServer(mux, listener)
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
