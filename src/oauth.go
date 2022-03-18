package eduvpn

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type OAuthGenStateUnableError struct {
	Err error
}

func (e *OAuthGenStateUnableError) Error() string {
	return fmt.Sprintf("failed generating state with error %v", e.Err)
}

type OAuthGenVerifierUnableError struct {
	Err error
}

func (e *OAuthGenVerifierUnableError) Error() string {
	return fmt.Sprintf("failed generating verifier with error %v", e.Err)
}

// Generates a random base64 string to be used for state
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// "state":  OPTIONAL.  An opaque value used by the client to maintain
// state between the request and callback.  The authorization server
// includes this value when redirecting the user agent back to the
// client.
func genState() (string, error) {
	randomBytes, err := MakeRandomByteSlice(32)

	if err != nil {
		return "", &OAuthGenStateUnableError{Err: err}
	}

	// For consistency we also use raw url encoding here
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// Generates a sha256 base64 challenge from a verifier
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.8
func genChallengeS256(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))

	// We use raw url encoding as the challenge does not accept padding
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// Generates a verifier
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// The code_verifier is a unique high-entropy cryptographically random
// string generated for each authorization request, using the unreserved
// characters [A-Z] / [a-z] / [0-9] / "-" / "." / "_" / "~", with a
// minimum length of 43 characters and a maximum length of 128
// characters.
func genVerifier() (string, error) {
	randomBytes, err := MakeRandomByteSlice(32)
	if err != nil {
		return "", &OAuthGenVerifierUnableError{Err: err}
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// This structure gets passed to the callback for easy access to the current state
type EduVPNOAuthSession struct {
	// Public
	AuthURL  string
	VPNState *EduVPNState

	// private
	callbackError error
	context       context.Context
	state         string
	server        *http.Server
	verifier      string
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type EduVPNOAuthToken struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Type    string `json:"token_type"`
	Expires int    `json:"expires_in"`
}

type OAuthFailedCallbackError struct {
	Addr string
	Err  error
}

func (e *OAuthFailedCallbackError) Error() string {
	return fmt.Sprintf("failed callback %s with error %v", e.Addr, e.Err)
}

// Gets an authenticated HTTP client by obtaining refresh and access tokens
func (eduvpn *EduVPNOAuthSession) getHTTPTokenClient() error {
	eduvpn.context = context.Background()
	mux := http.NewServeMux()
	addr := "127.0.0.1:8000"
	eduvpn.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	mux.HandleFunc("/callback", eduvpn.oauthCallback)
	if err := eduvpn.server.ListenAndServe(); err != http.ErrServerClosed {
		return &OAuthFailedCallbackError{Addr: addr, Err: err}
	}
	return eduvpn.callbackError
}

// Get the access and refresh tokens
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (eduvpn *EduVPNOAuthSession) getTokens(authCode string) error {
	// Make sure the verifier is set as the parameter
	// so that the server can verify that we are the actual owner of the authorization code

	reqURL := eduvpn.VPNState.Endpoints.API.V3.Token
	data := url.Values{
		"client_id":     {eduvpn.VPNState.Name},
		"code":          {authCode},
		"code_verifier": {eduvpn.verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"http://127.0.0.1:8000/callback"},
	}
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"}}
	opts := &HTTPOptionalParams{Headers: headers}
	body, bodyErr := HTTPPostWithOptionalParams(reqURL, data, opts)
	if bodyErr != nil {
		return bodyErr
	}

	tokenStructure := &EduVPNOAuthToken{}
	jsonErr := json.Unmarshal(body, tokenStructure)

	if jsonErr != nil {
		return &HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr}
	}

	eduvpn.VPNState.OAuthToken = tokenStructure

	return nil
}

type OAuthFailedCallbackParameterError struct {
	Parameter string
	URL       string
}

func (e *OAuthFailedCallbackParameterError) Error() string {
	return fmt.Sprintf("failed retrieving parameter %s in url %s", e.Parameter, e.URL)
}

type OAuthFailedCallbackStateMatchError struct {
	State         string
	ExpectedState string
}

func (e *OAuthFailedCallbackStateMatchError) Error() string {
	return fmt.Sprintf("failed matching state, got %s, want %s", e.State, e.ExpectedState)
}

type OAuthFailedCallbackGetTokensError struct {
	Err error
}

func (e *OAuthFailedCallbackGetTokensError) Error() string {
	return fmt.Sprintf("failed getting tokens with error %v", e.Err)
}

//
//// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (eduvpn *EduVPNOAuthSession) oauthCallback(w http.ResponseWriter, req *http.Request) {
	// Extract the authorization code
	code, success := req.URL.Query()["code"]
	if !success {
		eduvpn.callbackError = &OAuthFailedCallbackParameterError{Parameter: "code", URL: req.URL.String()}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}
	// The code is the first entry
	extractedCode := code[0]

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	state, success := req.URL.Query()["state"]
	if !success {
		eduvpn.callbackError = &OAuthFailedCallbackParameterError{Parameter: "state", URL: req.URL.String()}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}
	// The state is the first entry
	extractedState := state[0]
	if extractedState != eduvpn.state {
		eduvpn.callbackError = &OAuthFailedCallbackStateMatchError{State: extractedState, ExpectedState: eduvpn.state}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	err := eduvpn.getTokens(extractedCode)

	if err != nil {
		eduvpn.callbackError = &OAuthFailedCallbackGetTokensError{Err: err}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}

	// Shutdown the server as we're done listening
	go eduvpn.server.Shutdown(eduvpn.context)
}

func constructURL(baseURL string, parameters map[string]string) (string, error) {
	url, err := url.Parse(baseURL)

	if err != nil {
		return "", err
	}

	q := url.Query()

	for parameter, value := range parameters {
		q.Set(parameter, value)
	}
	url.RawQuery = q.Encode()
	return url.String(), nil
}

type OAuthFailedInitializeError struct {
	Err error
}

func (e *OAuthFailedInitializeError) Error() string {
	return fmt.Sprintf("failed initializing OAuth with error %v", e.Err)
}

// Initializes the OAuth for eduvpn.
// It needs a vpn state that was gotten from `Register`
// It returns the authurl for the browser and an error if present
func InitializeOAuth(vpnState *EduVPNState) (string, error) {
	if vpnState == nil {
		panic("invalid state")
	}

	// Generate the state
	state, stateErr := genState()
	if stateErr != nil {
		return "", &OAuthFailedInitializeError{Err: stateErr}
	}

	// Generate the verifier and challenge
	verifier, verifierErr := genVerifier()
	if verifierErr != nil {
		return "", &OAuthFailedInitializeError{Err: verifierErr}
	}
	challenge := genChallengeS256(verifier)

	parameters := map[string]string{
		"client_id":             vpnState.Name,
		"code_challenge_method": "S256",
		"code_challenge":        challenge,
		"response_type":         "code",
		"scope":                 "config",
		"state":                 state,
		"redirect_uri":          "http://127.0.0.1:8000/callback",
	}

	authURL, urlErr := constructURL(vpnState.Endpoints.API.V3.Authorization, parameters)

	if urlErr != nil { // shouldn't happen
		panic(urlErr)
	}

	// Fill the struct with the necessary fields filled for the next call to getting the HTTP client
	vpnState.OAuthSession = &EduVPNOAuthSession{AuthURL: authURL, VPNState: vpnState, state: state, verifier: verifier}
	return authURL, nil
}

func FinishOAuth(vpnState *EduVPNState) error {
	if vpnState == nil {
		panic("invalid state")
	}

	if vpnState.OAuthSession == nil {
		panic("invalid oauth state")
	}
	return vpnState.OAuthSession.getHTTPTokenClient()
}
