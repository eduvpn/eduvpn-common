package eduvpn

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"golang.org/x/oauth2"
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
		return "", err
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
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}


// This structure gets passed to the callback for easy access to the current state
type EduVPNOauth struct {
	// Public
	AuthURL string
	Config *oauth2.Config

	// private
	callbackError error
	client *http.Client
	context context.Context
	state string
	server *http.Server
	verifier string
}

// Initializes the OAuth eduvpn class. It returns a tuple of the class and error.
// If the error is non-nil, the class will be nil.
func InitializeOAuth(config *oauth2.Config) (*EduVPNOauth, error) {
	// Generate the state
	state, stateErr := genState()
	if stateErr != nil {
		return nil, detailedOAuthError{errGenStateError, fmt.Sprintf("oauth failed to gen random bytes for state"), stateErr}
	}

	// Generate the verifier and challenge
	verifier, err := genVerifier()
	if err != nil {
		return nil, detailedOAuthError{errGenVerifierError, fmt.Sprintf("oauth failed to verifier"), err}
	}
	challenge := genChallengeS256(verifier)

	// Update the auth url with the challenge and state
	codeChallengeMethod := oauth2.SetAuthURLParam("code_challenge_method", "S256")
	codeChallenge := oauth2.SetAuthURLParam("code_challenge", challenge)
	authURL := config.AuthCodeURL(state, codeChallengeMethod, codeChallenge)

	// Return the struct with the necessary fields filled for the next call to getting the HTTP client
	return &EduVPNOauth{AuthURL: authURL, Config: config, state: state, verifier: verifier}, nil
}

// Gets an authenticated HTTP client by obtaining refresh and access tokens
func (eduvpn* EduVPNOauth) GetHTTPTokenClient() (*http.Client, error) {
	eduvpn.context = context.Background()
	mux := http.NewServeMux()
	eduvpn.server = &http.Server {
		Addr: "127.0.0.1:8000",
		Handler: mux,
	}
	mux.HandleFunc("/callback", eduvpn.oauthCallback)
	if err := eduvpn.server.ListenAndServe(); err != http.ErrServerClosed {
		return nil, detailedOAuthError{errCallbackServerError, fmt.Sprintf("oauth callback server error"), err}
	}
	return eduvpn.client, eduvpn.callbackError
}



// Get the access and refresh tokens
// Access tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.4
// Refresh tokens: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.2
func (eduvpn *EduVPNOauth) getTokens(authCode string) error {
	// Make sure the verifier is set as the parameter
	// so that the server can verify that we are the actual owner of the authorization code
	codeVerifier := oauth2.SetAuthURLParam("code_verifier", eduvpn.verifier)

	// This is from the oauth library that gets the tokens from the authcode
	// https://pkg.go.dev/golang.org/x/oauth2#Config.Exchange
	// We pass the verifier as an additional parameter
	tok, err := eduvpn.Config.Exchange(eduvpn.context, authCode, codeVerifier)
	if err != nil {
		return err
	}

	// Fill the http client for future use in the struct
	eduvpn.client = eduvpn.Config.Client(eduvpn.context, tok)
	return nil
}

// The callback to retrieve the authorization code: https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-1.3.1
func (eduvpn *EduVPNOauth) oauthCallback(w http.ResponseWriter, req *http.Request) {
	// Extract the authorization code
	code, success := req.URL.Query()["code"]
	if !success {
		eduvpn.callbackError = detailedOAuthError{errCallbackGetAuthCodeError, fmt.Sprintf("oauth auth code cannot be retrieved"), nil}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}
	// The code is the first entry
	extractedCode := code[0]


	// Make sure the state is present and matches to protect against cross-site request forgeries
        // https://datatracker.ietf.org/doc/html/draft-ietf-oauth-v2-1-04#section-7.15
	state, success := req.URL.Query()["state"]
	if !success {
		eduvpn.callbackError = detailedOAuthError{errCallbackGetStateError, fmt.Sprintf("oauth state cannot be retrieved"), nil}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}
	// The state is the first entry
	extractedState := state[0]
	if extractedState != eduvpn.state {
		eduvpn.callbackError = detailedOAuthError{errCallbackVerifyStateMatchError, fmt.Sprintf("oauth state does not match"), nil}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}

	// Now that we have obtained the authorization code, we can move to the next step:
	// Obtaining the access and refresh tokens
	err := eduvpn.getTokens(extractedCode)

	if err != nil {
		eduvpn.callbackError = detailedOAuthError{errCallbackGetTokenExchangeError, fmt.Sprintf("oauth failed to get token in exchange"), err}
		go eduvpn.server.Shutdown(eduvpn.context)
		return
	}

	// Shutdown the server as we're done listening
	go eduvpn.server.Shutdown(eduvpn.context)
}

// OAuthErrorCode Simplified error code for public interface.
type OAuthErrorCode = VPNErrorCode
type OAuthError = VPNError

// detailedOAuthErrorCode used for unit tests.
type detailedOAuthErrorCode = detailedVPNErrorCode
type detailedOAuthError = detailedVPNError

const (
	ErrGenError OAuthErrorCode = iota + 1
	ErrCallbackTokenError
)

const (
	errGenStateError detailedOAuthErrorCode = iota + 1
	errGenVerifierError
	errCallbackServerError
	errCallbackGetAuthCodeError
	errCallbackGetStateError
	errCallbackVerifyStateMatchError
	errCallbackGetTokenExchangeError
)

func (err detailedOAuthError) ToOAuthError() OAuthError {
	return RequestError{err.Code.ToOAuthErrorCode(), err}
}

func (code detailedOAuthErrorCode) ToOAuthErrorCode() OAuthErrorCode {
	switch code {
	case errGenStateError:
	case errGenVerifierError:
		return ErrGenError

	case errCallbackServerError:
	case errCallbackGetAuthCodeError:
	case errCallbackGetStateError:
	case errCallbackVerifyStateMatchError:
	case errCallbackGetTokenExchangeError:
		return ErrCallbackTokenError
	}
	panic("invalid detailedOAuthErrorCode")
}
