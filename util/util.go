// package util defines public utility functions to be used by applications
// these are outside of the client package as they can be used even if a client hasn't been created yet
package util

import (
	"net"
	"strings"

	"github.com/eduvpn/eduvpn-common/i18nerr"
)

// CalculateGateway takes a CIDR encoded subnet `cidr` and returns the gateway and an error
func CalculateGateway(cidr string) (string, error) {
	_, ipn, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", i18nerr.WrapInternalf(err, "failed to parse CIDR for calculating gateway: %v", cidr)
	}

	ret := make(net.IP, len(ipn.IP))
	copy(ret, ipn.IP)

	for i := len(ret) - 1; i >= 0; i-- {
		ret[i]++
		if ret[i] > 0 {
			break
		}
	}

	if !ipn.Contains(ret) {
		return "", i18nerr.Newf("IP network does not contain incremented IP: %v", ret)
	}

	return ret.String(), nil
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
