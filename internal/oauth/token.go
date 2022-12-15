package oauth

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-errors/errors"
)

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

// token is a structure that contains our access and refresh tokens and a timestamp when they expire.
type token struct {
	// Access is the access token returned by the server
	access string

	// Refresh token is the refresh token returned by the server
	refresh string

	// ExpiredTimestamp is the Expires field but converted to a Go timestamp
	expiredTimestamp time.Time

	// Refresher is the function that refreshes the token
	Refresher func(string) (*TokenResponse, time.Time, error)
}

// tokenLock is a wrapper around token that protects it with a lock
type tokenLock struct {
	// Protects t
	mu sync.Mutex

	// The token fields protected by the lock
	t *token
}

// Access gets the OAuth access token used for contacting the server API
// It returns the access token as a string, possibly obtained fresh using the refresher
// If the token cannot be obtained, an error is returned and the token is an empty string.
func (l *tokenLock) Access() (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// The tokens are not expired yet
	// So they should be valid, re-login not neede
	if !l.expired() {
		return l.t.access, nil
	}

	// Check if refresh is even possible by doing a simple check if the refresh token is empty
	// This is not needed but reduces API calls to the server
	if l.t.refresh == "" {
		return "", errors.Wrap(&TokensInvalidError{Cause: "no refresh token is present"}, 0)
	}

	// Otherwise refresh and then later return the access token if we are successful
	tr, s, err := l.t.Refresher(l.t.refresh)
	if err != nil {
		// We have failed to ensure the tokens due to refresh not working
		return "", errors.Wrap(
			&TokensInvalidError{Cause: fmt.Sprintf("tokens failed refresh with error: %v", err)}, 0)
	}
	if tr == nil {
		return "", errors.New("No token response after refreshing")
	}
	l.updateInternal(*tr, s)
	return l.t.access, nil
}

// Clear completely clears the token structure
// This is useful for forcing re-authorization
func (l *tokenLock) Clear() {
	l.mu.Lock()
	l.t = &token{}
	l.mu.Unlock()
}

// updateInternal updates the structure using the response without locking
func (l *tokenLock) updateInternal(r TokenResponse, s time.Time) {
	l.t.access = r.Access
	l.t.refresh = r.Refresh
	l.t.expiredTimestamp = s.Add(time.Second * time.Duration(r.Expires))
}

// Update updates the structure usign the response and locks
func (l *tokenLock) Update(r TokenResponse, s time.Time) {
	l.mu.Lock()
	l.updateInternal(r, s)
	l.mu.Unlock()
}

// SetExpired overrides the timestamp to the current time
// This marks the tokens as expired
func (l *tokenLock) SetExpired() {
	l.mu.Lock()
	l.t.expiredTimestamp = time.Now()
	l.mu.Unlock()
}

// expired checks if the access token is expired.
// This is only called internally and thus does not lock
func (l *tokenLock) expired() bool {
	now := time.Now()
	return !now.Before(l.t.expiredTimestamp)
}
