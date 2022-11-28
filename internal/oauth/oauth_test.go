package oauth

import (
	"net/url"
	"testing"
)

func Test_verifiergen(t *testing.T) {
	verifier, verifierErr := genVerifier()
	if verifierErr != nil {
		t.Fatalf("Gen verifier error: %v", verifierErr)
	}

	// Verifier must be at minimum 43 and at max 128 characters...
	// However... Our verifier is exactly 43!
	if len(verifier) != 43 {
		t.Fatalf(
			"Got verifier length: %d, want a verifier with at least 43 characters",
			len(verifier),
		)
	}

	_, unescapeErr := url.QueryUnescape(verifier)
	if unescapeErr != nil {
		t.Fatalf("Verifier: %s can not be unescaped", verifier)
	}
}
