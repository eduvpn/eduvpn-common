package oauth

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
	"time"
)

func Test_verifiergen(t *testing.T) {
	v, err := genVerifier()
	if err != nil {
		t.Fatalf("Gen verifier error: %v", err)
	}

	// Verifier must be at minimum 43 and at max 128 characters...
	// However... Our verifier is exactly 43!
	if len(v) != 43 {
		t.Fatalf(
			"Got verifier length: %d, want a verifier with at least 43 characters",
			len(v),
		)
	}

	_, err = url.QueryUnescape(v)
	if err != nil {
		t.Fatalf("Verifier: %s can not be unescaped", v)
	}
}

func Test_stategen(t *testing.T) {
	s1, err := genState()
	if err != nil {
		t.Fatalf("Error when generating state 1: %v", err)
	}

	s2, err := genState()
	if err != nil {
		t.Fatalf("Error when generating state 2: %v", err)
	}

	if s1 == s2 {
		t.Fatalf("State: %v, equal to: %v", s1, s2)
	}
}

func Test_challengergen(t *testing.T) {
	verifier := "test"
	// Calculated using: base64.urlsafe_b64encode(hashlib.sha256("test".encode("utf-8")).digest()).decode("utf-8").replace("=", "") in Python
	// This test might not be the best because we're now comparing two different implementations, but at least it gives us a way to see if we messed something up in a commit
	want := "n4bQgYhMfWWaL-qgxVrQFaO_TxsrC4Is0V1sFbDwCgg"
	got := genChallengeS256(verifier)

	if got != want {
		t.Fatalf("Challenger not equal, got: %v, want: %v", got, want)
	}
}

func Test_accessToken(t *testing.T) {
	o := OAuth{}
	_, err := o.AccessToken()
	if err == nil {
		t.Fatalf("No error when getting access token on empty structure")
	}

	// Here we should get no error because the access token is set and is not expired
	want := "test"
	expired := time.Now().Add(1 * time.Hour)
	o = OAuth{token: &tokenLock{t: &tokenRefresher{Token: Token{Access: want, ExpiredTimestamp: expired}}}}
	got, err := o.AccessToken()
	if err != nil {
		t.Fatalf("Got error when getting access token on non-empty structure: %v", err)
	}
	if got != want {
		t.Fatalf("Access token not equal, Got: %v, Want: %v", got, want)
	}

	// Set the tokens as expired
	o.SetTokenExpired()

	// We should not get an error because expired and no refresh token
	_, err = o.AccessToken()
	if err == nil {
		t.Fatal("Got no error when getting access token on non-empty structure and expired")
	}

	want = "test2"
	// Now we internally update the refresh function and refresh token, we should get new tokens
	refresh := "refresh"
	o.token.t.Refresh = refresh
	o.token.t.Refresher = func(refreshToken string) (*TokenResponse, time.Time, error) {
		if refreshToken != refresh {
			t.Fatalf("Passed refresh token to refresher not equal to updated refresh token, got: %v, want: %v", refreshToken, refresh)
		}
		// Only the access and refresh fields are really important
		r := &TokenResponse{Access: want, Refresh: "test2"}
		return r, expired, nil
	}

	got, err = o.AccessToken()
	if err != nil {
		t.Fatalf("Got error when getting access token on non-empty expired structure and with a 'valid' refresh token: %v", err)
	}
	if got != want {
		t.Fatalf("Access token not equal, Got: %v, Want: %v", got, want)
	}
}

func Test_secretJSON(t *testing.T) {
	// Access and refresh tokens should not be present in marshalled JSON
	a := "ineedtobesecret_access"
	r := "ineedtobesecret_refresh"
	o := OAuth{token: &tokenLock{t: &tokenRefresher{Token: Token{Access: a, Refresh: r}}}}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("Error when marshalling OAuth JSON: %v", err)
	}
	s := string(b)
	// Of course this is a very dumb check, it could be that we are writing in some other serialized format. However, we simply marshal the structure directly. Go just serializes this as a simple string
	if strings.Contains(s, a) {
		t.Fatalf("Serialized OAuth contains Access Token! Serialized: %v, Access Token: %v", s, a)
	}

	if strings.Contains(s, r) {
		t.Fatalf("Serialized OAuth contains Refresh Token! Serialized: %v, Refresh Token: %v", s, a)
	}
}
