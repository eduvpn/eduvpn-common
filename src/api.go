package eduvpn

import (
	"fmt"
	"net/http"
	"net/url"
)

// Authenticated wrappers on top of HTTP
func (eduvpn *VPNState) apiAuthenticatedWithOpts(method string, endpoint string, opts *HTTPOptionalParams) (http.Header, []byte, error) {
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &HTTPOptionalParams{}
	}
	url := eduvpn.Server.Endpoints.API.V3.API + endpoint

	// Ensure we have non-expired tokens
	oauthErr := eduvpn.EnsureTokensOAuth()

	if oauthErr != nil {
		return nil, nil, oauthErr
	}

	headerKey := "Authorization"
	headerValue := fmt.Sprintf("Bearer %s", eduvpn.Server.OAuth.Token.Access)
	if opts.Headers != nil {
		opts.Headers.Add(headerKey, headerValue)
	} else {
		opts.Headers = &http.Header{headerKey: {headerValue}}
	}
	header, body, bodyErr := HTTPMethodWithOpts(method, url, opts)
	if bodyErr != nil {
		return header, nil, bodyErr
	}
	return header, body, nil
}

func (eduvpn *VPNState) APIConnectWireguard(pubkey string) (string, string, error) {
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	urlForm := url.Values{
		"profile_id": {"default"},
		"public_key": {pubkey},
	}
	header, body, bodyErr := eduvpn.apiAuthenticatedWithOpts(http.MethodPost, "/connect", &HTTPOptionalParams{Headers: headers, Body: urlForm})
	if bodyErr != nil {
		return "", "", bodyErr
	}

	expires := header.Get("expires")
	return string(body), expires, nil
}
