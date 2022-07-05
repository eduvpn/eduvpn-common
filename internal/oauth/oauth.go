package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	httpw "github.com/jwijenbergh/eduvpn-common/internal/http"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
)

// Generates a random base64 string to be used for state
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// "state":  OPTIONAL.  An opaque value used by the client to maintain
// state between the request and callback.  The authorization server
// includes this value when redirecting the user agent back to the
// client.
func genState() (string, error) {
	randomBytes, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", &types.WrappedErrorMessage{Message: "failed generating an OAuth state", Err: err}
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
	randomBytes, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", &types.WrappedErrorMessage{Message: "failed generating an OAuth verifier", Err: err}
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

type OAuth struct {
	Session              OAuthExchangeSession `json:"-"`
	Token                OAuthToken           `json:"token"`
	BaseAuthorizationURL string               `json:"base_authorization_url"`
	TokenURL             string               `json:"token_url"`
	Logger               *log.FileLogger      `json:"-"`
	FSM                  *fsm.FSM             `json:"-"`
}

// This structure gets passed to the callback for easy access to the current state
type OAuthExchangeSession struct {
	// returned from the callback
	CallbackError error

	// filled in in initialize
	ClientID string
	State    string
	Verifier string

	// filled in when constructing the callback
	Context context.Context
	Server  http.Server
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type OAuthToken struct {
	Access           string `json:"access_token"`
	Refresh          string `json:"refresh_token"`
	Type             string `json:"token_type"`
	Expires          int64  `json:"expires_in"`
	ExpiredTimestamp int64  `json:"expires_in_timestamp"`
}

// Gets an authorized HTTP client by obtaining refresh and access tokens
func (oauth *OAuth) getTokensWithCallback() error {
	oauth.Session.Context = context.Background()
	mux := http.NewServeMux()
	addr := "127.0.0.1:8000"
	oauth.Session.Server = http.Server{
		Addr:    addr,
		Handler: mux,
	}
	mux.HandleFunc("/callback", oauth.Callback)
	if err := oauth.Session.Server.ListenAndServe(); err != http.ErrServerClosed {
		return &types.WrappedErrorMessage{Message: "failed getting tokens with callback", Err: err}
	}
	return oauth.Session.CallbackError
}

// Get the access and refresh tokens
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (oauth *OAuth) getTokensWithAuthCode(authCode string) error {
	errorMessage := "failed getting tokens with the authorization code"
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
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.HTTPOptionalParams{Headers: headers, Body: data}
	current_time := util.GenerateTimeSeconds()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: bodyErr}
	}

	tokenStructure := OAuthToken{}

	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: &httpw.HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr}}
	}

	tokenStructure.ExpiredTimestamp = current_time + tokenStructure.Expires
	oauth.Token = tokenStructure
	return nil
}

func (oauth *OAuth) isTokensExpired() bool {
	expired_time := oauth.Token.ExpiredTimestamp
	current_time := util.GenerateTimeSeconds()
	return current_time >= expired_time
}

// Get the access and refresh tokens with a previously received refresh token
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (oauth *OAuth) getTokensWithRefresh() error {
	errorMessage := "failed getting tokens with the refresh token"
	reqURL := oauth.TokenURL
	data := url.Values{
		"refresh_token": {oauth.Token.Refresh},
		"grant_type":    {"refresh_token"},
	}
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.HTTPOptionalParams{Headers: headers, Body: data}
	current_time := util.GenerateTimeSeconds()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: bodyErr}
	}

	tokenStructure := OAuthToken{}
	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: &httpw.HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr}}
	}

	tokenStructure.ExpiredTimestamp = current_time + tokenStructure.Expires
	oauth.Token = tokenStructure
	return nil
}

//
//// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (oauth *OAuth) Callback(w http.ResponseWriter, req *http.Request) {
	errorMessage := "failed callback to retrieve the authorization code"
	// Extract the authorization code
	code, success := req.URL.Query()["code"]
	// Shutdown after we're done
	defer func() {
		go oauth.Session.Server.Shutdown(oauth.Session.Context)
	}()
	if !success {
		oauth.Session.CallbackError = &types.WrappedErrorMessage{Message: errorMessage, Err: &OAuthCallbackParameterError{Parameter: "code", URL: req.URL.String()}}
		return
	}
	// The code is the first entry
	extractedCode := code[0]

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	state, success := req.URL.Query()["state"]

	if !success {
		oauth.Session.CallbackError = &types.WrappedErrorMessage{Message: errorMessage, Err: &OAuthCallbackParameterError{Parameter: "state", URL: req.URL.String()}}
		return
	}
	// The state is the first entry
	extractedState := state[0]
	if extractedState != oauth.Session.State {
		oauth.Session.CallbackError = &types.WrappedErrorMessage{Message: errorMessage, Err: &OAuthCallbackStateMatchError{State: extractedState, ExpectedState: oauth.Session.State}}
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	getTokensErr := oauth.getTokensWithAuthCode(extractedCode)
	if getTokensErr != nil {
		oauth.Session.CallbackError = &types.WrappedErrorMessage{Message: errorMessage, Err: getTokensErr}
		return
	}
}

func (oauth *OAuth) Update(fsm *fsm.FSM, logger *log.FileLogger) {
	oauth.FSM = fsm
	oauth.Logger = logger
}

func (oauth *OAuth) Init(baseAuthorizationURL string, tokenURL string, fsm *fsm.FSM, logger *log.FileLogger) {
	oauth.BaseAuthorizationURL = baseAuthorizationURL
	oauth.TokenURL = tokenURL
	oauth.FSM = fsm
	oauth.Logger = logger
}

// Starts the OAuth exchange for eduvpn.
func (oauth *OAuth) start(name string) error {
	errorMessage := "failed starting OAuth exchange"
	if !oauth.FSM.HasTransition(fsm.OAUTH_STARTED) {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: fsm.WrongStateTransitionError{Got: oauth.FSM.Current, Want: fsm.OAUTH_STARTED}.CustomError()}
	}
	// Generate the state
	state, stateErr := genState()
	if stateErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: stateErr}
	}

	// Generate the verifier and challenge
	verifier, verifierErr := genVerifier()
	if verifierErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: verifierErr}
	}
	challenge := genChallengeS256(verifier)

	parameters := map[string]string{
		"client_id":             name,
		"code_challenge_method": "S256",
		"code_challenge":        challenge,
		"response_type":         "code",
		"scope":                 "config",
		"state":                 state,
		"redirect_uri":          "http://127.0.0.1:8000/callback",
	}

	authURL, urlErr := httpw.HTTPConstructURL(oauth.BaseAuthorizationURL, parameters)

	if urlErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
	}

	// Fill the struct with the necessary fields filled for the next call to getting the HTTP client
	oauthSession := OAuthExchangeSession{ClientID: name, State: state, Verifier: verifier}
	oauth.Session = oauthSession
	// Run the state callback in the background so that the user can login while we start the callback server
	oauth.FSM.GoTransitionWithData(fsm.OAUTH_STARTED, authURL, true)
	return nil
}

// Error definitions
func (oauth *OAuth) Finish() error {
	errorMessage := "failed finishing OAuth"
	if !oauth.FSM.HasTransition(fsm.AUTHORIZED) {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: fsm.WrongStateTransitionError{Got: oauth.FSM.Current, Want: fsm.AUTHORIZED}.CustomError()}
	}
	tokenErr := oauth.getTokensWithCallback()

	if tokenErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: tokenErr}
	}
	oauth.FSM.GoTransition(fsm.AUTHORIZED)
	return nil
}

func (oauth *OAuth) Cancel() {
	oauth.Session.CallbackError = &types.WrappedErrorMessage{Message: "failed cancelling OAuth", Err: &OAuthCancelledCallbackError{}}
	oauth.Session.Server.Shutdown(oauth.Session.Context)
}

func (oauth *OAuth) Login(name string) error {
	errorMessage := "failed OAuth login"
	authInitializeErr := oauth.start(name)

	if authInitializeErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: authInitializeErr}
	}

	oauthErr := oauth.Finish()

	if oauthErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: oauthErr}
	}
	return nil
}

func (oauth *OAuth) NeedsRelogin() bool {
	// Access Token or Refresh Tokens empty, definitely needs a relogin
	if oauth.Token.Access == "" || oauth.Token.Refresh == "" {
		oauth.Logger.Log(log.LOG_INFO, "OAuth: Tokens are empty")
		return true
	}

	// We have tokens...

	// The tokens are not expired yet
	// No relogin is needed
	if !oauth.isTokensExpired() {
		oauth.Logger.Log(log.LOG_INFO, "OAuth: Tokens are not expired, re-login not needed")
		return false
	}

	refreshErr := oauth.getTokensWithRefresh()
	// We have obtained new tokens with refresh
	if refreshErr == nil {
		oauth.Logger.Log(log.LOG_INFO, "OAuth: Tokens could be re-acquired using the refresh token, re-login not needed")
		return false
	}

	// Otherwise relogin is really needed
	return true
}

type OAuthCancelledCallbackError struct{}

func (e *OAuthCancelledCallbackError) Error() string {
	return fmt.Sprintf("client cancelled OAuth")
}

type OAuthCallbackParameterError struct {
	Parameter string
	URL       string
}

func (e *OAuthCallbackParameterError) Error() string {
	return fmt.Sprintf("failed retrieving parameter: %s in url: %s", e.Parameter, e.URL)
}

type OAuthCallbackStateMatchError struct {
	State         string
	ExpectedState string
}

func (e *OAuthCallbackStateMatchError) Error() string {
	return fmt.Sprintf("failed matching state, got: %s, want: %s", e.State, e.ExpectedState)
}