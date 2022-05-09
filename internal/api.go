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
func apiAuthorized(server Server, method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &HTTPOptionalParams{}
	}
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return nil, nil, baseErr
	}

	url := base.Endpoints.API.V3.API + endpoint

	// Ensure we have valid tokens
	stateBefore := base.FSM.Current
	oauthErr := EnsureTokens(server)

	// we reset the state so that we go from the authorized state to the state we want
	base.FSM.Current = stateBefore

	if oauthErr != nil {
		return nil, nil, oauthErr
	}

	headerKey := "Authorization"
	headerValue := fmt.Sprintf("Bearer %s", server.GetOAuth().Token.Access)
	if opts.Headers != nil {
		opts.Headers.Add(headerKey, headerValue)
	} else {
		opts.Headers = http.Header{headerKey: {headerValue}}
	}
	return HTTPMethodWithOpts(method, url, opts)
}

func apiAuthorizedRetry(server Server, method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	header, body, bodyErr := apiAuthorized(server, method, endpoint, opts)
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return nil, nil, &APIAuthorizedError{Err: baseErr}
	}
	if bodyErr != nil {
		var error *HTTPStatusError

		// Only retry authorized if we get a HTTP 401
		if errors.As(bodyErr, &error) && error.Status == 401 {
			base.Logger.Log(LOG_INFO, fmt.Sprintf("API: Got HTTP error %v, retrying authorized", error))
			// Tell the method that the token is expired
			server.GetOAuth().Token.ExpiredTimestamp = GenerateTimeSeconds()
			retryHeader, retryBody, retryErr := apiAuthorized(server, method, endpoint, opts)
			if retryErr != nil {
				return nil, nil, &APIAuthorizedError{Err: retryErr}
			}
			return retryHeader, retryBody, nil
		}
		return nil, nil, &APIAuthorizedError{Err: bodyErr}
	}
	return header, body, nil
}

func APIInfo(server Server) error {
	_, body, bodyErr := apiAuthorizedRetry(server, http.MethodGet, "/info", nil)
	if bodyErr != nil {
		return &APIInfoError{Err: bodyErr}
	}
	structure := ServerProfileInfo{}
	jsonErr := json.Unmarshal(body, &structure)

	if jsonErr != nil {
		return &APIInfoError{Err: jsonErr}
	}

	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &APIInfoError{Err: baseErr}
	}

	// Store the profiles and make sure that the current profile is not overwritten
	previousProfile := base.Profiles.Current
	base.Profiles = structure
	base.Profiles.Current = previousProfile
	base.ProfilesRaw = string(body)
	return nil
}

func APIConnectWireguard(server Server, profile_id string, pubkey string) (string, string, error) {
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
		"public_key": {pubkey},
	}
	header, connectBody, connectErr := apiAuthorizedRetry(server, http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", &APIConnectWireguardError{Err: connectErr}
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}

func APIConnectOpenVPN(server Server, profile_id string) (string, string, error) {
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-openvpn-profile"},
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
	}
	header, connectBody, connectErr := apiAuthorizedRetry(server, http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if connectErr != nil {
		return "", "", &APIConnectOpenVPNError{Err: connectErr}
	}

	expires := header.Get("expires")
	return string(connectBody), expires, nil
}

// This needs no further return value as it's best effort
func APIDisconnect(server Server) {
	apiAuthorizedRetry(server, http.MethodPost, "/disconnect", nil)
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
