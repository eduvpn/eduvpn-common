package client

import (
	"strings"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	discotypes "github.com/eduvpn/eduvpn-common/types/discovery"
)

func (c *Client) hasDiscovery() bool {
	// see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php
	return strings.HasPrefix(c.Name, "org.eduvpn.app")
}

// DiscoOrganizations gets the organizations list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#organization-list.
func (c *Client) DiscoOrganizations(ck *cookie.Cookie) (orgs *discotypes.Organizations, err error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
	}

	orgs, err = c.cfg.Discovery().Organizations(ck.Context())
	if err != nil {
		err = i18nerr.Wrap(err, "An error occurred after getting the discovery files for the list of organizations")
	}
	return
}

// DiscoServers gets the servers list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#server-list.
func (c *Client) DiscoServers(ck *cookie.Cookie) (dss *discotypes.Servers, err error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
	}

	dss, err = c.cfg.Discovery().Servers(ck.Context())
	if err != nil {
		err = i18nerr.Wrap(err, "An error occurred after getting the discovery files for the list of servers")
	}
	return
}
