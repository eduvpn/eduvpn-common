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

	wk := "/.well-known/vpn-user-portal"

	u.Path = path.Join(u.Path, wk)
	_, body, err := httpw.Get(u.String())
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
	b, err := srv.Base()
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, "failed API authorized", 0)
	}

	// Join the paths
	u, err := url.Parse(b.Endpoints.API.V3.API)
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, "failed API authorized", 0)
	}
	u.Path = path.Join(u.Path, endpoint)

	// Make sure the tokens are valid, this will return an error if re-login is needed
	t, err := HeaderToken(srv)
	if err != nil {
		return nil, nil, errors.WrapPrefix(err, "failed API authorized", 0)
	}

	key := "Authorization"
	val := fmt.Sprintf("Bearer %s", t)
	if opts.Headers != nil {
		opts.Headers.Add(key, val)
	} else {
		opts.Headers = http.Header{key: {val}}
	}
	return httpw.MethodWithOpts(method, u.String(), opts)
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
	pi := ProfileInfo{}
	if err = json.Unmarshal(body, &pi); err != nil {
		return errors.WrapPrefix(err, "failed API /info", 0)
	}

	b, err := srv.Base()
	if err != nil {
		return err
	}

	// Store the profiles and make sure that the current profile is not overwritten
	prev := b.Profiles.Current
	b.Profiles = pi
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

	ptm, err := http.ParseTime(exp)
	if err != nil {
		return "", "", time.Time{}, errors.WrapPrefix(err, "failed obtaining a WireGuard configuration", 0)
	}

	ct := h.Get("content-type")
	c := "openvpn"
	if ct == "application/x-wireguard-profile" {
		c = "wireguard"
	}

	return string(body), c, ptm, nil
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

	exp := h.Get("expires")
	ptm, err := http.ParseTime(exp)
	if err != nil {
		return "", time.Time{}, errors.WrapPrefix(err, "failed obtaining an OpenVPN configuration", 0)
	}

	return string(body), ptm, nil
}

// APIDisconnect disconnects from the API.
// This needs no further return value as it's best effort.
func APIDisconnect(server Server) {
	_, _, _ = apiAuthorized(server, http.MethodPost, "/disconnect", nil)
}
