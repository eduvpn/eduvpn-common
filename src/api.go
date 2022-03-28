package eduvpn

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// Authenticated wrappers on top of HTTP
func (server *Server) apiAuthenticated(method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &HTTPOptionalParams{}
	}
	url := server.Endpoints.API.V3.API + endpoint

	// Ensure we have non-expired tokens
	oauthErr := server.OAuth.EnsureTokens()

	if oauthErr != nil {
		return nil, nil, oauthErr
	}

	headerKey := "Authorization"
	headerValue := fmt.Sprintf("Bearer %s", server.OAuth.Token.Access)
	if opts.Headers != nil {
		opts.Headers.Add(headerKey, headerValue)
	} else {
		opts.Headers = &http.Header{headerKey: {headerValue}}
	}
	return HTTPMethodWithOpts(method, url, opts)
}

func (server *Server) apiAuthenticatedRetry(method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	header, body, bodyErr := server.apiAuthenticated(method, endpoint, opts)
	if bodyErr != nil {
		var error *HTTPStatusError

		if errors.As(bodyErr, &error) {
			// Tell the method that the token is expired
			server.OAuth.Token.ExpiredTimestamp = GenerateTimeSeconds()
			return server.apiAuthenticated(method, endpoint, opts)
		}
		return header, nil, bodyErr
	}
	return header, body, bodyErr
}

func (server *Server) APIInfo() error {
	_, body, bodyErr := server.apiAuthenticatedRetry(http.MethodGet, "/info", nil)
	if bodyErr != nil {
		return bodyErr
	}
	structure := &ServerProfileInfo{}
	jsonErr := json.Unmarshal(body, structure)

	if jsonErr != nil {
		return jsonErr
	}

	server.Profiles = structure

	// FIXME: Implement profile selection callback
	server.Profiles.Current = 0
	return nil
}

func (server *Server) APIConnectWireguard(profile_id string, pubkey string) (string, string, error) {
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	urlForm := url.Values{
		"profile_id": {"default"},
		"public_key": {pubkey},
	}
	header, connectBody, connectErr := server.apiAuthenticatedRetry(http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", connectErr
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}

func (server *Server) APIConnectOpenVPN(profile_id string) (string, string, error) {
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-openvpn-profile"},
	}

	urlForm := url.Values{
		"profile_id": {"default"},
	}
	header, connectBody, connectErr := server.apiAuthenticatedRetry(http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", connectErr
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}
