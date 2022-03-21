package eduvpn

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

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

type OAuth struct {
	Session *OAuthExchangeSession
	Token *OAuthToken
	TokenURL string
}

// This structure gets passed to the callback for easy access to the current state
type OAuthExchangeSession struct {
	// returned from the callback
	CallbackError error

	// filled in in initialize
	ClientID  string
	State         string
	Verifier      string

	// filled in when constructing the callback
	Context       context.Context
	Server        *http.Server
}

func generateTimeSeconds() int64 {
	current := time.Now()
	return current.Unix()
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type OAuthToken struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
	Type    string `json:"token_type"`
	Expires int64  `json:"expires_in"`
	ExpiredTimestamp int64
}

// Gets an authenticated HTTP client by obtaining refresh and access tokens
func (oauth *OAuth) getTokensWithCallback() error {
	oauth.Session.Context = context.Background()
	mux := http.NewServeMux()
	addr := "127.0.0.1:8000"
	oauth.Session.Server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	mux.HandleFunc("/callback", oauth.Callback)
	if err := oauth.Session.Server.ListenAndServe(); err != http.ErrServerClosed {
		return &OAuthFailedCallbackError{Addr: addr, Err: err}
	}
	return oauth.Session.CallbackError
}

// Get the access and refresh tokens
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (oauth *OAuth) getTokensWithAuthCode(authCode string) error {
	// Make sure the verifier is set as the parameter
	// so that the server can verify that we are the actual owner of the authorization code

	reqURL := oauth.TokenURL
	data := url.Values{
		"client_id":     {oauth.Session.ClientID},
		"code":          {authCode},
		"code_verifier": {oauth.Session.Verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"http://127.0.0.1:8000/callback"},
	}
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"}}
	opts := &HTTPOptionalParams{Headers: headers}
	current_time := generateTimeSeconds()
	body, bodyErr := HTTPPostWithOptionalParams(reqURL, data, opts)
	if bodyErr != nil {
		return bodyErr
	}

	tokenStructure := &OAuthToken{}
	jsonErr := json.Unmarshal(body, tokenStructure)

	if jsonErr != nil {
		return &HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr}
	}

	tokenStructure.ExpiredTimestamp = current_time + tokenStructure.Expires
	oauth.Token = tokenStructure

	return nil
}

func (oauth *OAuth) isTokensExpired() bool {
	expired_time := oauth.Token.ExpiredTimestamp
	current_time := generateTimeSeconds()
	return current_time >= expired_time
}

// Get the access and refresh tokens with a previously received refresh token
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (oauth *OAuth) getTokensWithRefresh() error {
	reqURL := oauth.TokenURL
	data := url.Values{
		"refresh_token": {oauth.Token.Refresh},
		"grant_type":    {"refresh_token"},
	}
	headers := &http.Header{
		"content-type": {"application/x-www-form-urlencoded"}}
	opts := &HTTPOptionalParams{Headers: headers}
	current_time := generateTimeSeconds()
	body, bodyErr := HTTPPostWithOptionalParams(reqURL, data, opts)
	if bodyErr != nil {
		return bodyErr
	}

	tokenStructure := &OAuthToken{}
	jsonErr := json.Unmarshal(body, tokenStructure)

	if jsonErr != nil {
		return &HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr}
	}

	tokenStructure.ExpiredTimestamp = current_time + tokenStructure.Expires
	oauth.Token = tokenStructure

	return nil
}

//
//// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (oauth *OAuth) Callback(w http.ResponseWriter, req *http.Request) {
	// Extract the authorization code
	code, success := req.URL.Query()["code"]
	if !success {
		oauth.Session.CallbackError = &OAuthFailedCallbackParameterError{Parameter: "code", URL: req.URL.String()}
		go oauth.Session.Server.Shutdown(oauth.Session.Context)
		return
	}
	// The code is the first entry
	extractedCode := code[0]

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	state, success := req.URL.Query()["state"]
	if !success {
		oauth.Session.CallbackError = &OAuthFailedCallbackParameterError{Parameter: "state", URL: req.URL.String()}
		go oauth.Session.Server.Shutdown(oauth.Session.Context)
		return
	}
	// The state is the first entry
	extractedState := state[0]
	if extractedState != oauth.Session.State {
		oauth.Session.CallbackError = &OAuthFailedCallbackStateMatchError{State: extractedState, ExpectedState: oauth.Session.State}
		go oauth.Session.Server.Shutdown(oauth.Session.Context)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	err := oauth.getTokensWithAuthCode(extractedCode)

	if err != nil {
		oauth.Session.CallbackError = &OAuthFailedCallbackGetTokensError{Err: err}
		go oauth.Session.Server.Shutdown(oauth.Session.Context)
		return
	}

	// Shutdown the server as we're done listening
	go oauth.Session.Server.Shutdown(oauth.Session.Context)
}

// Initializes the OAuth for eduvpn.
// It needs a vpn state that was gotten from `Register`
// It returns the authurl for the browser and an error if present
func (eduvpn *VPNState) InitializeOAuth() (string, error) {
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
		"client_id":             eduvpn.Name,
		"code_challenge_method": "S256",
		"code_challenge":        challenge,
		"response_type":         "code",
		"scope":                 "config",
		"state":                 state,
		"redirect_uri":          "http://127.0.0.1:8000/callback",
	}

	authURL, urlErr := HTTPConstructURL(eduvpn.Server.Endpoints.API.V3.Authorization, parameters)

	if urlErr != nil { // shouldn't happen
		panic(urlErr)
	}

	// Fill the struct with the necessary fields filled for the next call to getting the HTTP client
	oauthSession := &OAuthExchangeSession{ClientID: eduvpn.Name, State: state, Verifier: verifier}
	eduvpn.Server.OAuth = &OAuth{TokenURL: eduvpn.Server.Endpoints.API.V3.Token, Session: oauthSession}
	return authURL, nil
}


// Error definitions
func (eduvpn *VPNState) FinishOAuth() error {
	oauth := eduvpn.Server.OAuth
	if oauth == nil {
		panic("invalid oauth state")
	}
	return oauth.getTokensWithCallback()
}

func (eduvpn *VPNState) EnsureTokensOAuth() error {
	oauth := eduvpn.Server.OAuth
	if oauth == nil {
		panic("invalid oauth state")
	}

	if oauth.isTokensExpired() {
		return oauth.getTokensWithRefresh();
	}
	return nil
}


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


type OAuthFailedCallbackError struct {
	Addr string
	Err  error
}

func (e *OAuthFailedCallbackError) Error() string {
	return fmt.Sprintf("failed callback %s with error %v", e.Addr, e.Err)
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

type OAuthFailedInitializeError struct {
	Err error
}

func (e *OAuthFailedInitializeError) Error() string {
	return fmt.Sprintf("failed initializing OAuth with error %v", e.Err)
}
