package util

import (
	"crypto/rand"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eduvpn/eduvpn-common/types"
)

func EnsureValidURL(s string) (string, error) {
	parsedURL, parseErr := url.Parse(s)
	if parseErr != nil {
		return "", &types.WrappedErrorMessage{
			Message: fmt.Sprintf("failed parsing url: %s", s),
			Err:     parseErr,
		}
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
		return &types.WrappedErrorMessage{
			Message: fmt.Sprintf("failed to create directory %s", directory),
			Err:     mkdirErr,
		}
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
	// We just return the authURL in the cases where the template is not given or is invalid
	if authTemplate == "" {
		return authURL
	}
	if !strings.Contains(authTemplate, "@RETURN_TO@") {
		return authURL
	}
	if !strings.Contains(authTemplate, "@ORG_ID@") {
		return authURL
	}
	// Replace authURL
	authTemplate = strings.Replace(authTemplate, "@RETURN_TO@", WAYFEncode(authURL), 1)

	// If now there is no more ORG_ID, return as there weren't enough @ symbols
	if !strings.Contains(authTemplate, "@ORG_ID@") {
		return authURL
	}
	// Replace ORG ID
	authTemplate = strings.Replace(authTemplate, "@ORG_ID@", WAYFEncode(orgID), 1)
	return authTemplate
}

// https://github.com/eduvpn/documentation/blob/dc4d53c47dd7a69e95d6650eec408e16eaa814a2/SERVER_DISCOVERY.md#language-matching
func GetLanguageMatched(languageMap map[string]string, languageTag string) string {
	// If no map is given, return the empty string
	if len(languageMap) == 0 {
		return ""
	}
	// Try to find the exact match
	if val, ok := languageMap[languageTag]; ok {
		return val
	}
	// Try to find a key that starts with the OS language setting
	for k := range languageMap {
		if strings.HasPrefix(k, languageTag) {
			return languageMap[k]
		}
	}
	// Try to find a key that starts with the first part of the OS language (e.g. de-)
	splitted := strings.Split(languageTag, "-")
	// We have a "-"
	if len(splitted) > 1 {
		for k := range languageMap {
			if strings.HasPrefix(k, splitted[0]+"-") {
				return languageMap[k]
			}
		}
	}
	// search for just the language (e.g. de)
	for k := range languageMap {
		if k == splitted[0] {
			return languageMap[k]
		}
	}

	// Pick one that is deemed best, e.g. en-US or en, but note that not all languages are always available!
	// We force an entry that is english exactly or with an english prefix
	for k := range languageMap {
		if k == "en" || strings.HasPrefix(k, "en-") {
			return languageMap[k]
		}
	}

	// Otherwise just return one
	for k := range languageMap {
		return languageMap[k]
	}

	return ""
}
