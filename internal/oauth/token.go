package oauth

import "time"

// OAuthTokenResponse defines the OAuth response from the server that includes the tokens.
type OAuthTokenResponse struct {
	// Access is the access token returned by the server
	Access           string    `json:"access_token"`

	// Refresh token is the refresh token returned by the server
	Refresh          string    `json:"refresh_token"`

	// Type indicates which type of tokens we have
	Type             string    `json:"token_type"`

	// Expires is the expires time returned by the server
	Expires          int64     `json:"expires_in"`

}

// OAuthToken is a structure that contains our access and refresh tokens and a timestamp when they expire.
type OAuthToken struct {
	// Access is the access token returned by the server
	access           string

	// Refresh token is the refresh token returned by the server
	refresh          string

	// ExpiredTimestamp is the Expires field but converted to a Go timestamp
	expiredTimestamp time.Time
}

// Expired checks if the access token is expired.
func (tokens *OAuthToken) Expired() bool {
	currentTime := time.Now()
	return !currentTime.Before(tokens.expiredTimestamp)
}
