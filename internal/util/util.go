package util

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

func EnsureValidURL(s string) (string, error) {
	parsedURL, parseErr := url.Parse(s)
	if parseErr != nil {
		return "", &types.WrappedErrorMessage{Message: fmt.Sprintf("failed parsing url: %s", s), Err: parseErr}
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}
	return parsedURL.String(), nil
}

// Creates a random byteslice of `size`
func MakeRandomByteSlice(size int) ([]byte, error) {
	byteSlice := make([]byte, size)
	_, err := rand.Read(byteSlice)
	if err != nil {
		return nil, &types.WrappedErrorMessage{Message: "failed reading random", Err: err}
	}
	return byteSlice, nil
}

func GetCurrentTime() time.Time {
	return time.Now()
}

func EnsureDirectory(directory string) error {
	// Create with 700 permissions, read, write, execute only for the owner
	mkdirErr := os.MkdirAll(directory, 0o700)
	if mkdirErr != nil {
		return &types.WrappedErrorMessage{Message: fmt.Sprintf("failed to create directory %s", directory), Err: mkdirErr}
	}
	return nil
}

// See https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY_SKIP_WAYF.md
// URL encode for skipping where are you from (WAYF). Note that this right now is basically an alias to QueryEscape
func WAYFEncode(input string) string {
	// QueryReplace already replaces a space with a +
	// see https://go.dev/play/p/pOfrn-Wsq5
	return url.QueryEscape(input)
}

// See https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY_SKIP_WAYF.md
func ReplaceWAYF(authTemplate string, authURL string, orgID string) string {
	if authTemplate == "" {
		return authURL
	}
	// Replace authURL
	authTemplate = strings.Replace(authTemplate, "@RETURN_TO@", WAYFEncode(authURL), 1)
	// Replace ORG ID
	authTemplate = strings.Replace(authTemplate, "@ORG_ID@", WAYFEncode(orgID), 1)
	return authTemplate
}
