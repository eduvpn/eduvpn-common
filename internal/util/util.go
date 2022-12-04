// Package util implements several utility functions that are used across the codebase
package util

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-errors/errors"
)

// EnsureValidURL ensures that the input URL is valid to be used internally
// It does the following
// - Sets the scheme to https if none is given
// - It 'cleans' up the path using path.Clean
// - It makes sure that the URL ends with a /
// It returns an error if the URL cannot be parsed.
func EnsureValidURL(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", errors.WrapPrefix(err, "failed parsing url", 0)
	}

	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Path != "" {
		// Clean the path
		// https://pkg.go.dev/path#Clean
		u.Path = path.Clean(u.Path)
	}

	str := u.String()

	// Make sure the URL ends with a /
	if str[len(str)-1:] != "/" {
		str += "/"
	}
	return str, nil
}

// MakeRandomByteSlice creates a cryptographically random bytes slice of `size`
// It returns the byte slice (or nil if error) and an error if it could not be generated.
func MakeRandomByteSlice(size int) ([]byte, error) {
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return nil, errors.WrapPrefix(err, "failed reading random", 0)
	}
	return bs, nil
}

// EnsureDirectory creates a directory with permission 700.
func EnsureDirectory(dir string) error {
	// Create with 700 permissions, read, write, execute only for the owner
	err := os.MkdirAll(dir, 0o700)
	if err != nil {
		return errors.WrapPrefix(err, fmt.Sprintf("failed to create directory '%s'", dir), 0)
	}
	return nil
}

// ReplaceWAYF replaces an authorization template containing of @RETURN_TO@ and @ORG_ID@ with the authorization URL and the organization ID
// See https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY_SKIP_WAYF.md
func ReplaceWAYF(authTplt string, authURL string, orgID string) string {
	// We just return the authURL in the cases where the template is not given or is invalid
	if authTplt == "" {
		return authURL
	}
	if !strings.Contains(authTplt, "@RETURN_TO@") {
		return authURL
	}
	if !strings.Contains(authTplt, "@ORG_ID@") {
		return authURL
	}
	// Replace authURL
	authTplt = strings.Replace(authTplt, "@RETURN_TO@", url.QueryEscape(authURL), 1)

	// If now there is no more ORG_ID, return as there weren't enough @ symbols
	if !strings.Contains(authTplt, "@ORG_ID@") {
		return authURL
	}
	// Replace ORG ID
	authTplt = strings.Replace(authTplt, "@ORG_ID@", url.QueryEscape(orgID), 1)
	return authTplt
}

// GetLanguageMatched uses a map from language tags to strings to extract the right language given the tag
// It implements it according to https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY.md#language-matching
func GetLanguageMatched(langMap map[string]string, langTag string) string {
	// If no map is given, return the empty string
	if len(langMap) == 0 {
		return ""
	}
	// Try to find the exact match
	if val, ok := langMap[langTag]; ok {
		return val
	}
	// Try to find a key that starts with the OS language setting
	for k := range langMap {
		if strings.HasPrefix(k, langTag) {
			return langMap[k]
		}
	}
	// Try to find a key that starts with the first part of the OS language (e.g. de-)
	pts := strings.Split(langTag, "-")
	// We have a "-"
	if len(pts) > 1 {
		for k := range langMap {
			if strings.HasPrefix(k, pts[0]+"-") {
				return langMap[k]
			}
		}
	}
	// search for just the language (e.g. de)
	for k := range langMap {
		if k == pts[0] {
			return langMap[k]
		}
	}

	// Pick one that is deemed best, e.g. en-US or en, but note that not all languages are always available!
	// We force an entry that is english exactly or with an english prefix
	for k := range langMap {
		if k == "en" || strings.HasPrefix(k, "en-") {
			return langMap[k]
		}
	}

	// Otherwise just return one
	for k := range langMap {
		return langMap[k]
	}

	return ""
}
