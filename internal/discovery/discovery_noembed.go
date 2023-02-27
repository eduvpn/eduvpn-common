// +build !release

package discovery

// eServers are the embedded server list
// In this case this is empty because we do not embed during development
var eServers []byte

// eOrganizations are the embedded organizations
// In this case this is empty because we do not embed during development
var eOrganizations []byte

func init() {
	HasCache = false
}
