package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/go-errors/errors"
)

func validateEndpoints(endpoints Endpoints) error {
	v3 := endpoints.API.V3
	pAPI, err := url.Parse(v3.API)
	if err != nil {
		return errors.WrapPrefix(err, "failed to parse API endpoint", 0)
	}
	pAuth, err := url.Parse(v3.Authorization)
	if err != nil {
		return errors.WrapPrefix(err, "failed to parse API authorization endpoint", 0)
	}
	pToken, err := url.Parse(v3.Token)
	if err != nil {
		return errors.WrapPrefix(err, "failed to parse API token endpoint", 0)
	}
	if pAPI.Scheme != pAuth.Scheme {
		return errors.Errorf("API scheme: '%v', is not equal to authorization scheme: '%v'", pAPI.Scheme, pAuth.Scheme)
	}
	if pAPI.Scheme != pToken.Scheme {
		return errors.Errorf("API scheme: '%v', is not equal to token scheme: '%v'", pAPI.Scheme, pToken.Scheme)
	}
	if pAPI.Host != pAuth.Host {
		return errors.Errorf("API host: '%v', is not equal to authorization host: '%v'", pAPI.Host, pAuth.Host)
	}
	if pAPI.Host != pToken.Host {
		return errors.Errorf("API host: '%v', is not equal to token host: '%v'", pAPI.Host, pToken.Host)
	}
	return nil
}

func APIGetEndpoints(baseURL string, client *httpw.Client) (*Endpoints, error) {
	uStr, err := httpw.JoinURLPath(baseURL, "/.well-known/vpn-user-portal")
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = httpw.NewClient()
	}
	_, body, err := client.Get(uStr)
	if err != nil {
		return nil, errors.WrapPrefix(err, "failed getting server endpoints", 0)
	}

	ep := Endpoints{}
	if err = json.Unmarshal(body, &ep); err != nil {
		return nil, errors.WrapPrefix(err, "failed getting server endpoints", 0)
	}
	err = validateEndpoints(ep)
	if err != nil {
		return nil, err
	}

	return &ep, nil
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
		log.Logger.Debugf("Got a 401 error after HTTP method: %s, endpoint: %s. Marking token as expired...", method, endpoint)
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
	// The timeout is a bit lower here such that this does not take a too long time for disconnecting
	// Clients may wish to retry this
	_, _, err := apiAuthorized(server, http.MethodPost, "/disconnect", &httpw.OptionalParams{Timeout: 5 * time.Second})
	return err
}
