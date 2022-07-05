package util

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"time"

	"net/url"

	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

// Creates a random byteslice of `size`
func MakeRandomByteSlice(size int) ([]byte, error) {
	byteSlice := make([]byte, size)
	_, err := rand.Read(byteSlice)
	if err != nil {
		return nil, &types.WrappedErrorMessage{Message: "failed reading random", Err: err}
	}
	return byteSlice, nil
}

func GenerateTimeSeconds() int64 {
	current := time.Now()
	return current.Unix()
}

func EnsureDirectory(directory string) error {
	mkdirErr := os.MkdirAll(directory, os.ModePerm)
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
