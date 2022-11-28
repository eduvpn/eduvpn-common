// package oauth implement an oauth client defined in e.g. rfc 6749
// However, we try to follow some recommendations from the v2.1 oauth draft RFC
// Some specific things we implement here:
// - PKCE (RFC 7636)
// - ISS (RFC 9207)
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

// genState generates a random base64 string to be used for state
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// "state":  OPTIONAL.  An opaque value used by the client to maintain
// state between the request and callback.  The authorization server
// includes this value when redirecting the user agent back to the
// client.
// We implement it similarly to the verifier.
func genState() (string, error) {
	randomBytes, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", types.NewWrappedError("failed generating an OAuth state", err)
	}

	// For consistency we also use raw url encoding here
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}

// genChallengeS256 generates a sha256 base64 challenge from a verifier
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.8
func genChallengeS256(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))

	// We use raw url encoding as the challenge does not accept padding
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// genVerifier generates a verifier
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

// OAuth defines the main structure for this package.
type OAuth struct {
	// ISS indicates the issuer indentifier of the authorization server as defined in RFC 9207
	ISS                  string               `json:"iss"`

	// Token is where the access and refresh tokens are stored along with the timestamps
	Token                OAuthToken           `json:"token"`

	// BaseAuthorizationURL is the URL where authorization should take place
	BaseAuthorizationURL string               `json:"base_authorization_url"`

	// TokenURL is the URL where tokens should be obtained
	TokenURL             string               `json:"token_url"`

	// session is the internal in progress OAuth session
	session              OAuthExchangeSession `json:"-"`
}

// OAuthExchangeSession is a structure that gets passed to the callback for easy access to the current state.
type OAuthExchangeSession struct {
	// CallbackError indicates an error returned by the server
	CallbackError error

	// ClientID is the ID of the OAuth client
	ClientID string

	// ISS indicates the issuer inditifer
	ISS      string

	// State is the expected URL state paremeter
	State    string

	// Verifier is the preimage of the challenge
	Verifier string

	// Context is the context used for cancellation
	Context  context.Context

	// Server is the server of the session
	Server   *http.Server

	// Listener is the listener where the servers 'listens' on
	Listener net.Listener
}

// OAuthToken is a structure that defines the json format for /.well-known/vpn-user-portal".
type OAuthToken struct {
	// Access is the access token returned by the server
	Access           string    `json:"access_token"`

	// Refresh token is the refresh token returned by the server
	Refresh          string    `json:"refresh_token"`

	// Type indicates which type of tokens we have
	Type             string    `json:"token_type"`

	// Expires is the expires time returned by the server
	Expires          int64     `json:"expires_in"`

	// ExpiredTimestamp is the Expires field but converted to a Go timestamp
	ExpiredTimestamp time.Time `json:"expires_in_timestamp"`
}

// setupListener sets up an OAuth listener
// If it was unsuccessful it returns an error.
func (oauth *OAuth) setupListener() error {
	errorMessage := "failed setting up listener"
	oauth.session.Context = context.Background()

	// create a listener
	listener, listenerErr := net.Listen("tcp", ":0")
	if listenerErr != nil {
		return types.NewWrappedError(errorMessage, listenerErr)
	}
	oauth.session.Listener = listener
	return nil
}

// tokensWithCallback gets the OAuth tokens using a local web server
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithCallback() error {
	errorMessage := "failed getting tokens with callback"
	if oauth.session.Listener == nil {
		return types.NewWrappedError(errorMessage, errors.New("no listener"))
	}
	mux := http.NewServeMux()
	// server /callback over the listener address
	oauth.session.Server = &http.Server{
		Handler: mux,
	}
	mux.HandleFunc("/callback", oauth.Callback)

	if err := oauth.session.Server.Serve(oauth.session.Listener); err != http.ErrServerClosed {
		return types.NewWrappedError(errorMessage, err)
	}
	return oauth.session.CallbackError
}

// tokensWithAuthCode gets the access and refresh tokens using the authorization code
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithAuthCode(authCode string) error {
	errorMessage := "failed getting tokens with the authorization code"
	// Make sure the verifier is set as the parameter
	// so that the server can verify that we are the actual owner of the authorization code
	reqURL := oauth.TokenURL

	port, portErr := oauth.ListenerPort()
	if portErr != nil {
		return types.NewWrappedError(errorMessage, portErr)
	}

	data := url.Values{
		"client_id":     {oauth.session.ClientID},
		"code":          {authCode},
		"code_verifier": {oauth.session.Verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {fmt.Sprintf("http://127.0.0.1:%d/callback", port)},
	}
	headers := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.HTTPOptionalParams{Headers: headers, Body: data}
	currentTime := time.Now()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return types.NewWrappedError(errorMessage, bodyErr)
	}

	tokenStructure := OAuthToken{}

	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return types.NewWrappedError(
			errorMessage,
			&httpw.HTTPParseJSONError{URL: reqURL, Body: string(body), Err: jsonErr},
		)
	}

	tokenStructure.ExpiredTimestamp = currentTime.Add(
		time.Second * time.Duration(tokenStructure.Expires),
	)
	oauth.Token = tokenStructure
	return nil
}

// isTokensExpired returns if the OAuth tokens are expired using the expired timestamp.
func (oauth *OAuth) isTokensExpired() bool {
	expiredTime := oauth.Token.ExpiredTimestamp
	currentTime := time.Now()
	return !currentTime.Before(expiredTime)
}

// tokensWithRefresh gets the access and refresh tokens with a previously received refresh token
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithRefresh() error {
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
	currentTime := time.Now()
	_, body, bodyErr := httpw.HTTPPostWithOpts(reqURL, opts)
	if bodyErr != nil {
		return types.NewWrappedError(errorMessage, bodyErr)
	}

	tokenStructure := OAuthToken{}
	jsonErr := json.Unmarshal(body, &tokenStructure)

	if jsonErr != nil {
		return types.NewWrappedError(
			errorMessage,
			&httpw.HTTPParseJSONError{URL: reqURL, Body: string(body), Err: jsonErr},
		)
	}

	tokenStructure.ExpiredTimestamp = currentTime.Add(
		time.Second * time.Duration(tokenStructure.Expires),
	)
	oauth.Token = tokenStructure
	return nil
}

// responseTemplate is the HTML template for the OAuth authorized response
// this template was dapted from: https://github.com/eduvpn/apple/blob/5b18f834be7aebfed00570ae0c2f7bcbaf1c69cc/EduVPN/Helpers/Mac/OAuthRedirectHTTPHandler.m#L25
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

// oauthResponseHTML is a structure that is used to give back the OAuth response.
type oauthResponseHTML struct {
	Title string
	Message string
}

// writeResponseHTML writes the OAuth response using a response writer and the title + message
// If it was unsuccessful it returns an error.
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

// Callback is the public function used to get the OAuth tokens using an authorization code callback
// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (oauth *OAuth) Callback(w http.ResponseWriter, req *http.Request) {
	errorMessage := "failed callback to retrieve the authorization code"

	// Shutdown after we're done
	defer func() {
		// writing the html is best effort
		if oauth.session.CallbackError != nil {
			_ = writeResponseHTML(w, "Authorization Failed", "The authorization has failed. See the log file for more information.")
		} else {
			_ = writeResponseHTML(w, "Authorized", "The client has been successfully authorized. You can close this browser window.")
		}
		if oauth.session.Server != nil {
			go oauth.session.Server.Shutdown(oauth.session.Context) //nolint:errcheck
		}
	}()

	// ISS: https://www.rfc-editor.org/rfc/rfc9207.html
	// TODO: Make this a required parameter in the future
	urlQuery := req.URL.Query()
	extractedISS := urlQuery.Get("iss")
	if extractedISS != "" {
		if oauth.session.ISS != extractedISS {
			oauth.session.CallbackError = types.NewWrappedError(
				errorMessage,
				&OAuthCallbackISSMatchError{ISS: extractedISS, ExpectedISS: oauth.session.ISS},
			)
			return
		}

	}

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	extractedState := urlQuery.Get("state")
	if extractedState == "" {
		oauth.session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackParameterError{Parameter: "state", URL: req.URL.String()},
		)
		return
	}
	// The state is the first entry
	if extractedState != oauth.session.State {
		oauth.session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackStateMatchError{
				State:         extractedState,
				ExpectedState: oauth.session.State,
			},
		)
		return
	}

	// No authorization code
	extractedCode := urlQuery.Get("code")
	if extractedCode == "" {
		oauth.session.CallbackError = types.NewWrappedError(
			errorMessage,
			&OAuthCallbackParameterError{Parameter: "code", URL: req.URL.String()},
		)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	getTokensErr := oauth.tokensWithAuthCode(extractedCode)
	if getTokensErr != nil {
		oauth.session.CallbackError = types.NewWrappedError(
			errorMessage,
			getTokensErr,
		)
		return
	}
}

// Init initializes OAuth with the following parameters:
// - OAuth server issuer identification
// - The URL used for authorization
// - The URL to obtain new tokens.
func (oauth *OAuth) Init(iss string, baseAuthorizationURL string, tokenURL string) {
	oauth.ISS = iss
	oauth.BaseAuthorizationURL = baseAuthorizationURL
	oauth.TokenURL = tokenURL
}

// ListenerPort gets the listener for the OAuth web server
// It returns the port as an integer and an error if there is any.
func (oauth OAuth) ListenerPort() (int, error) {
	errorMessage := "failed to get listener port"

	if oauth.session.Listener == nil {
		return 0, types.NewWrappedError(errorMessage, errors.New("no OAuth listener"))
	}
	return oauth.session.Listener.Addr().(*net.TCPAddr).Port, nil
}

// AuthURL gets the authorization url to start the OAuth procedure.
func (oauth *OAuth) AuthURL(name string, postProcessAuth func(string) string) (string, error) {
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
	oauth.session = oauthSession

	// set up the listener to get the redirect URI
	listenerErr := oauth.setupListener()
	if listenerErr != nil {
		return "", types.NewWrappedError(errorMessage, stateErr)
	}

	// Get the listener port
	port, portErr := oauth.ListenerPort()
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

// Exchange starts the OAuth exchange by getting the tokens with the redirect callback
// If it was unsuccessful it returns an error.
func (oauth *OAuth) Exchange() error {
	tokenErr := oauth.tokensWithCallback()

	if tokenErr != nil {
		return types.NewWrappedError("failed finishing OAuth", tokenErr)
	}
	return nil
}

// Cancel cancels the existing OAuth
// TODO: Use context for this.
func (oauth *OAuth) Cancel() {
	oauth.session.CallbackError = types.NewWrappedErrorLevel(
		types.ErrInfo,
		"cancelled OAuth",
		&OAuthCancelledCallbackError{},
	)
	if oauth.session.Server != nil {
		oauth.session.Server.Shutdown(oauth.session.Context) //nolint:errcheck
	}
}

// EnsureTokens makes sure the OAuth tokens are still valid
// if this cannot be guaranteed, it returns an error.
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
	refreshErr := oauth.tokensWithRefresh()
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
