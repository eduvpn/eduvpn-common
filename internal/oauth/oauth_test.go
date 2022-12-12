package oauth

import (
	"net/url"
	"testing"
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
