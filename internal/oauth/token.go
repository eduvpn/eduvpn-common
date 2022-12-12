package oauth

import "time"

// TokenResponse defines the OAuth response from the server that includes the tokens.
type TokenResponse struct {
	// Access is the access token returned by the server
	Access string `json:"access_token"`

	// Refresh token is the refresh token returned by the server
	Refresh string `json:"refresh_token"`

	// Type indicates which type of tokens we have
	Type string `json:"token_type"`

	// Expires is the expires time returned by the server
	Expires int64 `json:"expires_in"`
}

// Token is a structure that contains our access and refresh tokens and a timestamp when they expire.
type Token struct {
	// Access is the access token returned by the server
	access string

	// Refresh token is the refresh token returned by the server
	refresh string

	// ExpiredTimestamp is the Expires field but converted to a Go timestamp
	expiredTimestamp time.Time
}

// Expired checks if the access token is expired.
func (tokens *Token) Expired() bool {
	now := time.Now()
	return !now.Before(tokens.expiredTimestamp)
}
