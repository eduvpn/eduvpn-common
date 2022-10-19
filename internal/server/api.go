package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/types"
)

func APIGetEndpoints(baseURL string) (*ServerEndpoints, error) {
	errorMessage := "failed getting server endpoints"
	url, urlErr := url.Parse(baseURL)
	if urlErr != nil {
		return nil, types.NewWrappedError(errorMessage, urlErr)
	}

	url.Path = path.Join(url.Path, WellKnownPath)
	_, body, bodyErr := httpw.HTTPGet(url.String())

	if bodyErr != nil {
		return nil, types.NewWrappedError(errorMessage, bodyErr)
	}

	endpoints := &ServerEndpoints{}
	jsonErr := json.Unmarshal(body, endpoints)

	if jsonErr != nil {
		return nil, types.NewWrappedError(errorMessage, jsonErr)
	}

	return endpoints, nil
}

func apiAuthorized(
	server Server,
	method string,
	endpoint string,
	opts *httpw.HTTPOptionalParams,
) (http.Header, []byte, error) {
	errorMessage := "failed API authorized"
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &httpw.HTTPOptionalParams{}
	}
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, baseErr)
	}

	// Join the paths
	url, urlErr := url.Parse(base.Endpoints.API.V3.API)
	if urlErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, urlErr)
	}
	url.Path = path.Join(url.Path, endpoint)

	// Make sure the tokens are valid, this will return an error if re-login is needed
	oauthErr := EnsureTokens(server)
	if oauthErr != nil {
		return nil, nil, types.NewWrappedError(errorMessage, oauthErr)
	}

	headerKey := "Authorization"
	headerValue := fmt.Sprintf("Bearer %s", GetHeaderToken(server))
	if opts.Headers != nil {
		opts.Headers.Add(headerKey, headerValue)
	} else {
		opts.Headers = http.Header{headerKey: {headerValue}}
	}
	return httpw.HTTPMethodWithOpts(method, url.String(), opts)
}

func apiAuthorizedRetry(
	server Server,
	method string,
	endpoint string,
	opts *httpw.HTTPOptionalParams,
) (http.Header, []byte, error) {
	errorMessage := "failed authorized API retry"
	header, body, bodyErr := apiAuthorized(server, method, endpoint, opts)

	if bodyErr != nil {
		var error *httpw.HTTPStatusError

		// Only retry authorized if we get a HTTP 401
		if errors.As(bodyErr, &error) && error.Status == 401 {
			// Mark the token as expired and retry so we trigger the refresh flow
			MarkTokenExpired(server)
			retryHeader, retryBody, retryErr := apiAuthorized(server, method, endpoint, opts)
			if retryErr != nil {
				return nil, nil, types.NewWrappedError(errorMessage, retryErr)
			}
			return retryHeader, retryBody, nil
		}
		return nil, nil, types.NewWrappedError(errorMessage, bodyErr)
	}
	return header, body, nil
}

func APIInfo(server Server) error {
	errorMessage := "failed API /info"
	_, body, bodyErr := apiAuthorizedRetry(server, http.MethodGet, "/info", nil)
	if bodyErr != nil {
		return types.NewWrappedError(errorMessage, bodyErr)
	}
	structure := ServerProfileInfo{}
	jsonErr := json.Unmarshal(body, &structure)

	if jsonErr != nil {
		return types.NewWrappedError(errorMessage, jsonErr)
	}

	base, baseErr := server.GetBase()

	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	// Store the profiles and make sure that the current profile is not overwritten
	previousProfile := base.Profiles.Current
	base.Profiles = structure
	base.Profiles.Current = previousProfile
	return nil
}

// see https://github.com/eduvpn/documentation/blob/v3/API.md#request-1
func GetPreferTCPString(preferTCP bool) string {
	if preferTCP {
		return "yes"
	}
	return "no"
}

func APIConnectWireguard(
	server Server,
	profile_id string,
	pubkey string,
	preferTCP bool,
	supportsOpenVPN bool,
) (string, string, time.Time, error) {
	errorMessage := "failed obtaining a WireGuard configuration"
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	// This profile also supports OpenVPN
	// Indicate that we also accept OpenVPN profiles
	if supportsOpenVPN {
		headers.Add("accept", "application/x-openvpn-profile")
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
		"public_key": {pubkey},
		"prefer_tcp": {GetPreferTCPString(preferTCP)},
	}
	header, connectBody, connectErr := apiAuthorizedRetry(
		server,
		http.MethodPost,
		"/connect",
		&httpw.HTTPOptionalParams{Headers: headers, Body: urlForm},
	)
	if connectErr != nil {
		return "", "", time.Time{}, types.NewWrappedError(
			errorMessage,
			connectErr,
		)
	}

	expires := header.Get("expires")

	pTime, pTimeErr := http.ParseTime(expires)
	if pTimeErr != nil {
		return "", "", time.Time{}, types.NewWrappedError(errorMessage, pTimeErr)
	}

	contentType := header.Get("content-type")

	content := "openvpn"
	if contentType == "application/x-wireguard-profile" {
		content = "wireguard"
	}
	return string(connectBody), content, pTime, nil
}

func APIConnectOpenVPN(server Server, profile_id string, preferTCP bool) (string, time.Time, error) {
	errorMessage := "failed obtaining an OpenVPN configuration"
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-openvpn-profile"},
	}

	urlForm := url.Values{
		"profile_id": {profile_id},
		"prefer_tcp": {GetPreferTCPString(preferTCP)},
	}

	header, connectBody, connectErr := apiAuthorizedRetry(
		server,
		http.MethodPost,
		"/connect",
		&httpw.HTTPOptionalParams{Headers: headers, Body: urlForm},
	)
	if connectErr != nil {
		return "", time.Time{}, types.NewWrappedError(errorMessage, connectErr)
	}

	expires := header.Get("expires")
	pTime, pTimeErr := http.ParseTime(expires)
	if pTimeErr != nil {
		return "", time.Time{}, types.NewWrappedError(errorMessage, pTimeErr)
	}
	return string(connectBody), pTime, nil
}

// This needs no further return value as it's best effort
func APIDisconnect(server Server) {
	_, _, _ = apiAuthorized(server, http.MethodPost, "/disconnect", nil)
}
