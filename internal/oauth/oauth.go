// Package oauth implement an oauth client defined in e.g. rfc 6749
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
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/go-errors/errors"
)

// genState generates a random base64 string to be used for state
// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-4.1.1
// "state":  OPTIONAL.  An opaque value used by the client to maintain
// state between the request and callback.  The authorization server
// includes this value when redirecting the user agent back to the
// client.
// We implement it similarly to the verifier.
func genState() (string, error) {
	bs, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", err
	}

	// For consistency, we also use raw url encoding here
	return base64.RawURLEncoding.EncodeToString(bs), nil
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
//
//	NOTE: The code verifier SHOULD have enough entropy to make it
//	impractical to guess the value.  It is RECOMMENDED that the output of
//	a suitable random number generator be used to create a 32-octet
//	sequence.  The octet sequence is then base64url-encoded to produce a
//	43-octet URL safe string to use as the code verifier.
//
// See: https://datatracker.ietf.org/doc/html/rfc7636#section-4.1
func genVerifier() (string, error) {
	random, err := util.MakeRandomByteSlice(32)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(random), nil
}

// OAuth defines the main structure for this package.
type OAuth struct {
	// ISS indicates the issuer identifier of the authorization server as defined in RFC 9207
	ISS string `json:"iss"`

	// BaseAuthorizationURL is the URL where authorization should take place
	BaseAuthorizationURL string `json:"base_authorization_url"`

	// TokenURL is the URL where tokens should be obtained
	TokenURL string `json:"token_url"`

	// session is the internal in progress OAuth session
	session ExchangeSession

	// Token is where the access and refresh tokens are stored along with the timestamps
	token Token
}

// ExchangeSession is a structure that gets passed to the callback for easy access to the current state.
type ExchangeSession struct {
	// CallbackError indicates an error returned by the server
	CallbackError error

	// ClientID is the ID of the OAuth client
	ClientID string

	// ISS indicates the issuer identifier
	ISS string

	// State is the expected URL state parameter
	State string

	// Verifier is the preimage of the challenge
	Verifier string

	// Context is the context used for cancellation
	Context context.Context

	// Server is the server of the session
	Server *http.Server

	// Listener is the listener where the servers 'listens' on
	Listener net.Listener
}

// AccessToken gets the OAuth access token used for contacting the server API
// It returns the access token as a string, possibly obtained fresh using the Refresh Token
// If the token cannot be obtained, an error is returned and the token is an empty string.
func (oauth *OAuth) AccessToken() (string, error) {
	t := oauth.token

	// We have tokens...
	// The tokens are not expired yet
	// So they should be valid, re-login not needed
	if !t.Expired() {
		return t.access, nil
	}

	// Check if refresh is even possible by doing a simple check if the refresh token is empty
	// This is not needed but reduces API calls to the server
	if t.refresh == "" {
		return "", errors.Wrap(&TokensInvalidError{Cause: "no refresh token is present"}, 0)
	}

	// Otherwise refresh and then later return the access token if we are successful
	err := oauth.tokensWithRefresh()
	if err != nil {
		// We have failed to ensure the tokens due to refresh not working
		return "", errors.Wrap(
			&TokensInvalidError{Cause: fmt.Sprintf("tokens failed refresh with error: %v", err)}, 0)
	}

	// We have obtained new tokens with refresh
	return t.access, nil
}

// setupListener sets up an OAuth listener
// If it was unsuccessful it returns an error.
// @see https://www.ietf.org/archive/id/draft-ietf-oauth-v2-1-07.html#section-8.4.2
// "Loopback Interface Redirection".
func (oauth *OAuth) setupListener() error {
	oauth.session.Context = context.Background()

	// create a listener
	lst, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errors.WrapPrefix(err, "net.Listen failed", 0)
	}
	oauth.session.Listener = lst
	return nil
}

// tokensWithCallback gets the OAuth tokens using a local web server
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithCallback() error {
	if oauth.session.Listener == nil {
		return errors.Errorf("failed getting tokens with callback: no listener")
	}
	mux := http.NewServeMux()
	// server /callback over the listener address
	oauth.session.Server = &http.Server{
		Handler: mux,
		// Define a default 60 second header read timeout to protect against a Slowloris Attack
		// A bit overkill maybe for a local server but good to define anyways
		ReadHeaderTimeout: 60 * time.Second,
	}
	mux.HandleFunc("/callback", oauth.Callback)

	if err := oauth.session.Server.Serve(oauth.session.Listener); err != http.ErrServerClosed {
		return errors.WrapPrefix(err, "failed getting tokens with callback", 0)
	}
	return oauth.session.CallbackError
}

// fillToken fills the OAuth token structure by the response
// It calculates the expired timestamp by having a 'startTime' passed to it
// The URL that is input here is used for additional context.
func (oauth *OAuth) fillToken(response []byte, startTime time.Time, url string) error {
	res := TokenResponse{}

	err := json.Unmarshal(response, &res)
	if err != nil {
		return errors.WrapPrefix(err, "failed filling OAuth tokens from "+url, 0)
	}

	oauth.token = Token{
		access:           res.Access,
		refresh:          res.Refresh,
		expiredTimestamp: startTime.Add(time.Second * time.Duration(res.Expires)),
	}
	return nil
}

// SetTokenExpired marks the tokens as expired by setting the expired timestamp to the current time.
func (oauth *OAuth) SetTokenExpired() {
	oauth.token.expiredTimestamp = time.Now()
}

// SetTokenRenew sets the tokens for renewal by completely clearing the structure.
func (oauth *OAuth) SetTokenRenew() {
	oauth.token = Token{}
}

// tokensWithAuthCode gets the access and refresh tokens using the authorization code
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithAuthCode(authCode string) error {
	// Make sure the verifier is set as the parameter
	// so that the server can verify that we are the actual owner of the authorization code
	u := oauth.TokenURL

	port, err := oauth.ListenerPort()
	if err != nil {
		return err
	}

	data := url.Values{
		"client_id":     {oauth.session.ClientID},
		"code":          {authCode},
		"code_verifier": {oauth.session.Verifier},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {fmt.Sprintf("http://127.0.0.1:%d/callback", port)},
	}
	h := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.OptionalParams{Headers: h, Body: data}
	now := time.Now()
	_, body, err := httpw.PostWithOpts(u, opts)
	if err != nil {
		return err
	}

	return oauth.fillToken(body, now, u)
}

// tokensWithRefresh gets the access and refresh tokens with a previously received refresh token
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
// If it was unsuccessful it returns an error.
func (oauth *OAuth) tokensWithRefresh() error {
	u := oauth.TokenURL
	data := url.Values{
		"refresh_token": {oauth.token.refresh},
		"grant_type":    {"refresh_token"},
	}
	h := http.Header{
		"content-type": {"application/x-www-form-urlencoded"},
	}
	opts := &httpw.OptionalParams{Headers: h, Body: data}
	now := time.Now()
	_, body, err := httpw.PostWithOpts(u, opts)
	if err != nil {
		return err
	}

	return oauth.fillToken(body, now, u)
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
	Title   string
	Message string
}

// writeResponseHTML writes the OAuth response using a response writer and the title + message
// If it was unsuccessful it returns an error.
func writeResponseHTML(w http.ResponseWriter, title string, message string) error {
	t, err := template.New("oauth-response").Parse(responseTemplate)
	if err != nil {
		return errors.WrapPrefix(err, "failed writing response HTML", 0)
	}

	return t.Execute(w, oauthResponseHTML{Title: title, Message: message})
}

// Callback is the public function used to get the OAuth tokens using an authorization code callback
// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (oauth *OAuth) Callback(w http.ResponseWriter, req *http.Request) {
	// Shutdown after we're done
	defer func() {
		// writing the html is best effort
		if oauth.session.CallbackError != nil {
			_ = writeResponseHTML(
				w,
				"Authorization Failed",
				"The authorization has failed. See the log file for more information.",
			)
		} else {
			_ = writeResponseHTML(w, "Authorized", "The client has been successfully authorized. You can close this browser window.")
		}
		if oauth.session.Server != nil {
			go func() {
				_ = oauth.session.Server.Shutdown(oauth.session.Context) //nolint:errcheck
			}()
		}
	}()

	// ISS: https://www.rfc-editor.org/rfc/rfc9207.html
	// TODO: Make this a required parameter in the future
	q := req.URL.Query()
	iss := q.Get("iss")
	if iss != "" {
		if oauth.session.ISS != iss {
			oauth.session.CallbackError = errors.Errorf("failed matching ISS; expected '%s' got '%s'",
				oauth.session.ISS, iss)
			return
		}
	}

	// Make sure the state is present and matches to protect against cross-site request forgeries
	// https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	state := q.Get("state")
	if state == "" {
		oauth.session.CallbackError = errors.Errorf("failed retrieving parameter 'state' from '%s'", req.URL)
		return
	}
	// The state is the first entry
	if state != oauth.session.State {
		oauth.session.CallbackError = errors.Errorf("failed matching state; expected '%s' got '%s'",
			oauth.session.State, state)
		return
	}

	// No authorization code
	code := q.Get("code")
	if code == "" {
		oauth.session.CallbackError = errors.Errorf("failed retrieving parameter 'code' from '%s'", req.URL)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	if err := oauth.tokensWithAuthCode(code); err != nil {
		oauth.session.CallbackError = errors.WrapPrefix(err, "failed callback to retrieve the authorization code", 0)
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
func (oauth *OAuth) ListenerPort() (int, error) {
	if oauth.session.Listener == nil {
		return 0, errors.Errorf("failed to get listener port")
	}
	return oauth.session.Listener.Addr().(*net.TCPAddr).Port, nil
}

// AuthURL gets the authorization url to start the OAuth procedure.
func (oauth *OAuth) AuthURL(name string, postProcessAuth func(string) string) (string, error) {
	// Generate the verifier and challenge
	v, err := genVerifier()
	if err != nil {
		return "", errors.WrapPrefix(err, "genVerifier error", 0)
	}

	// Generate the state
	state, err := genState()
	if err != nil {
		return "", errors.WrapPrefix(err, "genState error", 0)
	}

	// Fill the struct with the necessary fields filled for the next call to getting the HTTP client
	oauth.session = ExchangeSession{
		ClientID: name,
		ISS:      oauth.ISS,
		State:    state,
		Verifier: v,
	}

	// set up the listener to get the redirect URI
	if err = oauth.setupListener(); err != nil {
		return "", errors.WrapPrefix(err, "oauth.setupListener error", 0)
	}

	// Get the listener port
	port, err := oauth.ListenerPort()
	if err != nil {
		return "", errors.WrapPrefix(err, "oauth.ListenerPort error", 0)
	}

	params := map[string]string{
		"client_id":             name,
		"code_challenge_method": "S256",
		"code_challenge":        genChallengeS256(v),
		"response_type":         "code",
		"scope":                 "config",
		"state":                 state,
		"redirect_uri":          fmt.Sprintf("http://127.0.0.1:%d/callback", port),
	}

	u, err := httpw.ConstructURL(oauth.BaseAuthorizationURL, params)

	if err != nil {
		return "", errors.WrapPrefix(err, "httpw.ConstructURL error", 0)
	}

	// Return the url processed
	return postProcessAuth(u), nil
}

// Exchange starts the OAuth exchange by getting the tokens with the redirect callback
// If it was unsuccessful it returns an error.
func (oauth *OAuth) Exchange() error {
	return oauth.tokensWithCallback()
}

// Cancel cancels the existing OAuth
// TODO: Use context for this.
func (oauth *OAuth) Cancel() {
	oauth.session.CallbackError = errors.Wrap(&CancelledCallbackError{}, 0)
	if oauth.session.Server != nil {
		_ = oauth.session.Server.Shutdown(oauth.session.Context) //nolint:errcheck
	}
}

type CancelledCallbackError struct{}

func (e *CancelledCallbackError) Error() string {
	return "client cancelled OAuth"
}

type TokensInvalidError struct {
	Cause string
}

func (e *TokensInvalidError) Error() string {
	return fmt.Sprintf("tokens are invalid due to: %s", e.Cause)
}
