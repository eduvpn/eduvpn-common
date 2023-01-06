package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/go-errors/errors"
)

func APIGetEndpoints(baseURL string) (*Endpoints, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, errors.WrapPrefix(err, "failed getting server endpoints", 0)
	}

	u.Path = path.Join(u.Path, "/.well-known/vpn-user-portal")
	c := httpw.NewClient()
	_, body, err := c.Get(u.String())
	if err != nil {
		return nil, errors.WrapPrefix(err, "failed getting server endpoints", 0)
	}

	ep := &Endpoints{}
	if err = json.Unmarshal(body, ep); err != nil {
		return nil, errors.WrapPrefix(err, "failed getting server endpoints", 0)
	}

	return ep, nil
}

func apiAuthorized(
	srv Server,
	method string,
	endpoint string,
	opts *httpw.OptionalParams,
) (http.Header, []byte, error) {
	// Ensure optional is not nil as we will fill it with headers
	if opts == nil {
		opts = &httpw.OptionalParams{}
	}
	errorMessage := "failed API authorized"
	b, err := srv.Base()
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, errorMessage, 0)
	}

	// Join the paths
	u, err := url.Parse(b.Endpoints.API.V3.API)
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, errorMessage, 0)
	}
	u.Path = path.Join(u.Path, endpoint)

	// Make sure the tokens are valid, this will return an error if re-login is needed
	t, err := HeaderToken(srv)
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, errorMessage, 0)
	}

	key := "Authorization"
	val := fmt.Sprintf("Bearer %s", t)
	if opts.Headers != nil {
		opts.Headers.Add(key, val)
	} else {
		opts.Headers = http.Header{key: {val}}
	}

	// Create a client if it doesn't exist
	if b.httpClient == nil {
		b.httpClient = httpw.NewClient()
	}
	return b.httpClient.Do(method, u.String(), opts)
}

func apiAuthorizedRetry(
	srv Server,
	method string,
	endpoint string,
	opts *httpw.OptionalParams,
) (http.Header, []byte, error) {
	h, body, err := apiAuthorized(srv, method, endpoint, opts)
	if err == nil {
		return h, body, nil
	}

	statErr := &httpw.StatusError{}
	// Only retry authorized if we get an HTTP 401
	if errors.As(err, &statErr) && statErr.Status == 401 {
		// Mark the token as expired and retry, so we trigger the refresh flow
		MarkTokenExpired(srv)
		h, body, err = apiAuthorized(srv, method, endpoint, opts)
	}
	return h, body, err
}

func APIInfo(srv Server) error {
	_, body, err := apiAuthorizedRetry(srv, http.MethodGet, "/info", nil)
	if err != nil {
		return err
	}
	profiles := ProfileInfo{}
	if err = json.Unmarshal(body, &profiles); err != nil {
		return errors.WrapPrefix(err, "failed API /info", 0)
	}

	b, err := srv.Base()
	if err != nil {
		return err
	}

	// Store the profiles and make sure that the current profile is not overwritten
	prev := b.Profiles.Current
	b.Profiles = profiles
	b.Profiles.Current = prev
	return nil
}

// see https://github.com/eduvpn/documentation/blob/v3/API.md#request-1
func boolToYesNo(preferTCP bool) string {
	if preferTCP {
		return "yes"
	}
	return "no"
}

func APIConnectWireguard(
	srv Server,
	profileID string,
	pubkey string,
	preferTCP bool,
	openVPNSupport bool,
) (string, string, time.Time, error) {
	hdrs := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-wireguard-profile"},
	}

	// This profile also supports OpenVPN
	// Indicate that we also accept OpenVPN profiles
	if openVPNSupport {
		hdrs.Add("accept", "application/x-openvpn-profile")
	}

	vals := url.Values{
		"profile_id": {profileID},
		"public_key": {pubkey},
		"prefer_tcp": {boolToYesNo(preferTCP)},
	}
	h, body, err := apiAuthorizedRetry(srv, http.MethodPost, "/connect",
		&httpw.OptionalParams{Headers: hdrs, Body: vals})
	if err != nil {
		return "", "", time.Time{}, err
	}

	exp := h.Get("expires")

	expTime, err := http.ParseTime(exp)
	if err != nil {
		return "", "", time.Time{}, errors.WrapPrefix(err, "failed obtaining a WireGuard configuration", 0)
	}

	contentH := h.Get("content-type")
	content := "openvpn"
	if contentH == "application/x-wireguard-profile" {
		content = "wireguard"
	}

	return string(body), content, expTime, nil
}

func APIConnectOpenVPN(srv Server, profileID string, preferTCP bool) (string, time.Time, error) {
	hdrs := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
		"accept":       {"application/x-openvpn-profile"},
	}

	vals := url.Values{
		"profile_id": {profileID},
		"prefer_tcp": {boolToYesNo(preferTCP)},
	}

	h, body, err := apiAuthorizedRetry(srv, http.MethodPost, "/connect",
		&httpw.OptionalParams{Headers: hdrs, Body: vals})
	if err != nil {
		return "", time.Time{}, err
	}

	expH := h.Get("expires")
	expT, err := http.ParseTime(expH)
	if err != nil {
		return "", time.Time{}, errors.WrapPrefix(err, "failed obtaining an OpenVPN configuration", 0)
	}

	return string(body), expT, nil
}

// APIDisconnect disconnects from the API.
func APIDisconnect(server Server) error {
	_, _, err := apiAuthorized(server, http.MethodPost, "/disconnect", nil)
	return err
}
