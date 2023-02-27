// +build release

package discovery

import _ "embed"

// eServers are the embedded server list
//
//go:embed server_list.json
var eServers []byte

// eOrganizations are the embedded organizations
//
//go:embed organization_list.json
var eOrganizations []byte

func init() {
	HasCache = true
}
