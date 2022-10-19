package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

// Generates a random base64 string to be used for state
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// "state":  OPTIONAL.  An opaque value used by the client to maintain
// state between the request and callback.  The authorization server
// includes this value when redirecting the user agent back to the
// client.
// We implement it similarly to the verifier
func genState() (string, error) {
	randomBytes, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", types.NewWrappedError("failed generating an OAuth state", err)
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
// We implement it according to the note:
//   NOTE: The code verifier SHOULD have enough entropy to make it
//   impractical to guess the value.  It is RECOMMENDED that the output of
//   a suitable random number generator be used to create a 32-octet
//   sequence.  The octet sequence is then base64url-encoded to produce a
//   43-octet URL safe string to use as the code verifier.
// See: https://datatracker.ietf.org/doc/html/rfc7636#section-4.1
func genVerifier() (string, error) {
	randomBytes, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", types.NewWrappedError(
			"failed generating an OAuth verifier",
			err,
		)
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

type OAuth struct {
	ISS                  string               `json:"iss"`
	Session              OAuthExchangeSession `json:"-"`
	Token                OAuthToken           `json:"token"`
	BaseAuthorizationURL string               `json:"base_authorization_url"`
	TokenURL             string               `json:"token_url"`
}

// This structure gets passed to the callback for easy access to the current state
type OAuthExchangeSession struct {
	// returned from the callback
	CallbackError error

	// filled in in initialize
	ClientID string
	ISS      string
	State    string
	Verifier string

	// filled in when constructing the callback
	Context  context.Context
	Server   *http.Server
	Listener net.Listener
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type OAuthToken struct {
	Access           string    `json:"access_token"`
	Refresh          string    `json:"refresh_token"`
	Type             string    `json:"token_type"`
	Expires          int64     `json:"expires_in"`
	ExpiredTimestamp time.Time `json:"expires_in_timestamp"`
}

// Sets up a listener
func (oauth *OAuth) setupListener() error {
	errorMessage := "failed setting up listener"
	oauth.Session.Context = context.Background()

	// create a listener
	listener, listenerErr := net.Listen("tcp", ":0")
	if listenerErr != nil {
		return types.NewWrappedError(errorMessage, listenerErr)
	}
	oauth.Session.Listener = listener
	return nil
}

func (oauth *OAuth) getTokensWithCallback() error {
	errorMessage := "failed getting tokens with callback"
	if oauth.Session.Listener == nil {
		return types.NewWrappedError(errorMessage, errors.New("No listener"))
	}
	mux := http.NewServeMux()
	// server /callback over the listener address
	oauth.Session.Server = &http.Server{
		Handler: mux,
	}
	mux.HandleFunc("/callback", oauth.Callback)

	if err := oauth.Session.Server.Serve(oauth.Session.Listener); err != http.ErrServerClosed {
		return types.NewWrappedError(errorMessage, err)
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

	port, portErr := oauth.GetListenerPort()
	if portErr != nil {
		return types.NewWrappedError(errorMessage, portErr)
	}

	data := url.Values{
		"client_id":     {oauth.Session.ClientID},
		"code":          {authCode},
		"code_verifier": {oauth.Session.Verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {fmt.Sprintf("http://127.0.0.1:%d/callback", port)},
	}
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.HTTPOptionalParams{Headers: headers, Body: data}
	current_time := util.GetCurrentTime()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return types.NewWrappedError(errorMessage, bodyErr)
	}

	tokenStructure := OAuthToken{}

	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return types.NewWrappedError(
			errorMessage,
			&httpw.HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr},
		)
	}

	tokenStructure.ExpiredTimestamp = current_time.Add(
		time.Second * time.Duration(tokenStructure.Expires),
	)
	oauth.Token = tokenStructure
	return nil
}

func (oauth *OAuth) isTokensExpired() bool {
	expired_time := oauth.Token.ExpiredTimestamp
	current_time := util.GetCurrentTime()
	return !current_time.Before(expired_time)
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
	current_time := util.GetCurrentTime()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return types.NewWrappedError(errorMessage, bodyErr)
	}

	tokenStructure := OAuthToken{}
	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return types.NewWrappedError(
			errorMessage,
			&httpw.HTTPParseJsonError{URL: reqURL, Body: string(body), Err: jsonErr},
		)
	}

	tokenStructure.ExpiredTimestamp = current_time.Add(
		time.Second * time.Duration(tokenStructure.Expires),
	)
	oauth.Token = tokenStructure
	return nil
}

// Adapted from: https://github.com/eduvpn/apple/blob/5b18f834be7aebfed00570ae0c2f7bcbaf1c69cc/EduVPN/Helpers/Mac/OAuthRedirectHTTPHandler.m#L25
const responseTemplate string = `
<!DOCTYPE html>
<html dir="ltr" xmlns="http://www.w3.org/1999/xhtml" lang="en"><head>
<meta http-equiv="content-type" content="text/html; charset=UTF-8">
<meta charset="utf-8">
<title>{{.Title}}</title>
<style>
body {
    font-family: arial;
    margin: 0;
    height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #ccc;
    color: #252622;
}
main {
    padding: 1em 2em;
    text-align: center;
    border: 2pt solid #666;
    box-shadow: rgba(0, 0, 0, 0.6) 0px 1px 4px;
    border-color: #aaa;
    background: #ddd;
}
</style>
</head>
<body>
    <main>
        <h1>{{.Title}}</h1>
        <p>{{.Message}}</p>
    </main>
</body>
</html>
`

type oauthResponseHTML struct {
	Title string
	Message string
}

func writeResponseHTML(w http.ResponseWriter, title string, message string) error {
	errorMessage := "failed writing response HTML"
	template, templateErr := template.New("oauth-response").Parse(responseTemplate)
	if templateErr != nil {
		return types.NewWrappedError(errorMessage, templateErr)
	}

	executeErr := template.Execute(w, oauthResponseHTML{
		Title:   title,
		Message: message,
	})
	if executeErr != nil {
		return types.NewWrappedError(errorMessage, executeErr)
	}
	return nil
}

//
//// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (oauth *OAuth) Callback(w http.ResponseWriter, req *http.Request) {
	errorMessage := "failed callback to retrieve the authorization code"

	// Shutdown after we're done
	defer func() {
		// writing the html is best effort
		if oauth.Session.CallbackError != nil {
			_ = writeResponseHTML(w, "Authorization Failed", "The authorization has failed. See the log file for more information.")
		} else {
			_ = writeResponseHTML(w, "Authorized", "The client has been successfully authorized. You can close this browser window.")
		}
		if oauth.Session.Server != nil {
			go oauth.Session.Server.Shutdown(oauth.Session.Context) //nolint:errcheck
		}
	}()

	// ISS: https://www.rfc-editor.org/rfc/rfc9207.html
	// TODO: Make this a required parameter in the future
	urlQuery := req.URL.Query()
	extractedISS := urlQuery.Get("iss")
	if extractedISS != "" {
		if oauth.Session.ISS != extractedISS {
			oauth.Session.CallbackError = types.NewWrappedError(
				errorMessage,
				&OAuthCallbackISSMatchError{ISS: extractedISS, ExpectedISS: oauth.Session.ISS},
			)
			return
		}

	}

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	extractedState := urlQuery.Get("state")
	if extractedState == "" {
		oauth.Session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackParameterError{Parameter: "state", URL: req.URL.String()},
		)
		return
	}
	// The state is the first entry
	if extractedState != oauth.Session.State {
		oauth.Session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackStateMatchError{
				State:         extractedState,
				ExpectedState: oauth.Session.State,
			},
		)
		return
	}

	// No authorization code
	extractedCode := urlQuery.Get("code")
	if extractedCode == "" {
		oauth.Session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackParameterError{Parameter: "code", URL: req.URL.String()},
		)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	getTokensErr := oauth.getTokensWithAuthCode(extractedCode)
	if getTokensErr != nil {
		oauth.Session.CallbackError = types.NewWrappedError(
			errorMessage,
			getTokensErr,
		)
		return
	}
}

func (oauth *OAuth) Init(iss string, baseAuthorizationURL string, tokenURL string) {
	oauth.ISS = iss
	oauth.BaseAuthorizationURL = baseAuthorizationURL
	oauth.TokenURL = tokenURL
}

func (oauth OAuth) GetListenerPort() (int, error) {
	errorMessage := "failed to get listener port"

	if oauth.Session.Listener == nil {
		return 0, types.NewWrappedError(errorMessage, errors.New("No OAuth listener"))
	}
	return oauth.Session.Listener.Addr().(*net.TCPAddr).Port, nil
}

// Starts the OAuth exchange for eduvpn.
func (oauth *OAuth) GetAuthURL(name string, postProcessAuth func(string) string) (string, error) {
	errorMessage := "failed starting OAuth exchange"

	// Generate the verifier and challenge
	verifier, verifierErr := genVerifier()
	if verifierErr != nil {
		return "", types.NewWrappedError(errorMessage, verifierErr)
	}
	challenge := genChallengeS256(verifier)

	// Generate the state
	state, stateErr := genState()
	if stateErr != nil {
		return "", types.NewWrappedError(errorMessage, stateErr)
	}

	// Fill the struct with the necessary fields filled for the next call to getting the HTTP client
	oauthSession := OAuthExchangeSession{ClientID: name, ISS: oauth.ISS, State: state, Verifier: verifier}
	oauth.Session = oauthSession

	// set up the listener to get the redirect URI
	listenerErr := oauth.setupListener()
	if listenerErr != nil {
		return "", types.NewWrappedError(errorMessage, stateErr)
	}

	// Get the listener port
	port, portErr := oauth.GetListenerPort()
	if portErr != nil {
		return "", types.NewWrappedError(errorMessage, portErr)
	}

	parameters := map[string]string{
		"client_id":             name,
		"code_challenge_method": "S256",
		"code_challenge":        challenge,
		"response_type":         "code",
		"scope":                 "config",
		"state":                 state,
		"redirect_uri":          fmt.Sprintf("http://127.0.0.1:%d/callback", port),
	}

	authURL, urlErr := httpw.HTTPConstructURL(oauth.BaseAuthorizationURL, parameters)

	if urlErr != nil {
		return "", types.NewWrappedError(errorMessage, urlErr)
	}

	// Return the url processed
	return postProcessAuth(authURL), nil
}

// Error definitions
func (oauth *OAuth) Exchange() error {
	tokenErr := oauth.getTokensWithCallback()

	if tokenErr != nil {
		return types.NewWrappedError("failed finishing OAuth", tokenErr)
	}
	return nil
}

func (oauth *OAuth) Cancel() {
	oauth.Session.CallbackError = types.NewWrappedErrorLevel(
		types.ERR_INFO,
		"cancelled OAuth",
		&OAuthCancelledCallbackError{},
	)
	if oauth.Session.Server != nil {
		oauth.Session.Server.Shutdown(oauth.Session.Context) //nolint:errcheck
	}
}

func (oauth *OAuth) EnsureTokens() error {
	errorMessage := "failed ensuring OAuth tokens"
	// Access Token or Refresh Tokens empty, we can not ensure the tokens
	if oauth.Token.Access == "" && oauth.Token.Refresh == "" {
		return types.NewWrappedError(
			errorMessage,
			&OAuthTokensInvalidError{Cause: "tokens are empty"},
		)
	}

	// We have tokens...
	// The tokens are not expired yet
	// So they should be valid, re-login not needed
	if !oauth.isTokensExpired() {
		return nil
	}

	// Otherwise try to refresh them and return if successful
	refreshErr := oauth.getTokensWithRefresh()
	// We have obtained new tokens with refresh
	if refreshErr != nil {
		// We have failed to ensure the tokens due to refresh not working
		return types.NewWrappedError(
			errorMessage,
			&OAuthTokensInvalidError{
				Cause: fmt.Sprintf("tokens failed refresh with error: %v", refreshErr),
			},
		)
	}

	return nil
}

type OAuthCancelledCallbackError struct{}

func (e *OAuthCancelledCallbackError) Error() string {
	return "client cancelled OAuth"
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

type OAuthCallbackISSMatchError struct {
	ISS         string
	ExpectedISS string
}

func (e *OAuthCallbackISSMatchError) Error() string {
	return fmt.Sprintf("failed matching ISS, got: %s, want: %s", e.ISS, e.ExpectedISS)
}

type OAuthTokensInvalidError struct {
	Cause string
}

func (e *OAuthTokensInvalidError) Error() string {
	return fmt.Sprintf("tokens are invalid due to: %s", e.Cause)
}
