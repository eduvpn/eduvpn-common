package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// Authorized wrappers on top of HTTP
// the errors will not be wrapped here so that the caller can check if we got a status error, to retry oauth
func (server *Server) apiAuthorized(method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &HTTPOptionalParams{}
	}
	url := server.Endpoints.API.V3.API + endpoint

	// Ensure we have valid tokens
	oauthErr := server.EnsureTokens()

	if oauthErr != nil {
		return nil, nil, oauthErr
	}

	headerKey := "Authorization"
	headerValue := fmt.Sprintf("Bearer %s", server.OAuth.Token.Access)
	if opts.Headers != nil {
		opts.Headers.Add(headerKey, headerValue)
	} else {
		opts.Headers = http.Header{headerKey: {headerValue}}
	}
	return HTTPMethodWithOpts(method, url, opts)
}

func (server *Server) apiAuthorizedRetry(method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	header, body, bodyErr := server.apiAuthorized(method, endpoint, opts)
	if bodyErr != nil {
		var error *HTTPStatusError

		// Only retry authorized if we get a HTTP 401
		if errors.As(bodyErr, &error) && error.Status == 401 {
			server.Logger.Log(LOG_INFO, fmt.Sprintf("API: Got HTTP error %v, retrying authorized", error))
			// Tell the method that the token is expired
			server.OAuth.Token.ExpiredTimestamp = GenerateTimeSeconds()
			retryHeader, retryBody, retryErr := server.apiAuthorized(method, endpoint, opts)
			if retryErr != nil {
				return nil, nil, &APIAuthorizedError{Err: retryErr}
			}
			return retryHeader, retryBody, nil
		}
		return nil, nil, &APIAuthorizedError{Err: bodyErr}
	}
	return header, body, nil
}

func (server *Server) APIInfo() error {
	_, body, bodyErr := server.apiAuthorizedRetry(http.MethodGet, "/info", nil)
	if bodyErr != nil {
		return &APIInfoError{Err: bodyErr}
	}
	structure := ServerProfileInfo{}
	jsonErr := json.Unmarshal(body, &structure)

	if jsonErr != nil {
		return &APIInfoError{Err: jsonErr}
	}

	server.Profiles = structure
	server.ProfilesRaw = string(body)
	return nil
}

func (server *Server) APIConnectWireguard(profile_id string, pubkey string) (string, string, error) {
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
		"public_key": {pubkey},
	}
	header, connectBody, connectErr := server.apiAuthorizedRetry(http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", &APIConnectWireguardError{Err: connectErr}
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}

func (server *Server) APIConnectOpenVPN(profile_id string) (string, string, error) {
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-openvpn-profile"},
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
	}
	header, connectBody, connectErr := server.apiAuthorizedRetry(http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", &APIConnectOpenVPNError{Err: connectErr}
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}

// This needs no further return value as it's best effort
func (server *Server) APIDisconnect() {
	server.apiAuthorizedRetry(http.MethodPost, "/disconnect", nil)
}

type APIAuthorizedError struct {
	Err error
}

func (e *APIAuthorizedError) Error() string {
	return fmt.Sprintf("failed api authorized call with error: %v", e.Err)
}

type APIConnectWireguardError struct {
	Err error
}

func (e *APIConnectWireguardError) Error() string {
	return fmt.Sprintf("failed api /connect wireguard call with error: %v", e.Err)
}

type APIConnectOpenVPNError struct {
	Err error
}

func (e *APIConnectOpenVPNError) Error() string {
	return fmt.Sprintf("failed api /connect OpenVPN call with error: %v", e.Err)
}

type APIInfoError struct {
	Err error
}

func (e *APIInfoError) Error() string {
	return fmt.Sprintf("failed api /info call with error: %v", e.Err)
}
