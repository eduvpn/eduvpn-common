// package util defines public utility functions to be used by applications
// these are outside of the client package as they can be used even if a client hasn't been created yet
package util

import (
	"net"

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
