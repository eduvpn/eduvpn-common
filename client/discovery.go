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
func (c *Client) DiscoOrganizations(ck *cookie.Cookie) (*discotypes.Organizations, error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
	}

	orgs, err := c.cfg.Discovery().Organizations(ck.Context())
	if err != nil {
		err = i18nerr.Wrap(err, "An error occurred after getting the discovery files for the list of organizations")
	}
	if orgs == nil {
		return nil, err
	}

	// convert to public subset
	retOrgs := make([]discotypes.Organization, len(orgs.List))
	for i, v := range orgs.List {
		retOrgs[i] = v.Organization
	}
	return &discotypes.Organizations{
		List: retOrgs,
	}, err
}

// DiscoServers gets the servers list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#server-list.
func (c *Client) DiscoServers(ck *cookie.Cookie) (*discotypes.Servers, error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
	}

	servs, err := c.cfg.Discovery().Servers(ck.Context())
	if err != nil {
		err = i18nerr.Wrap(err, "An error occurred after getting the discovery files for the list of servers")
	}
	if servs == nil {
		return nil, err
	}

	// convert to public subset
	retServs := make([]discotypes.Server, len(servs.List))
	for i, v := range servs.List {
		retServs[i] = v.Server
	}
	return &discotypes.Servers{
		List: retServs,
	}, err
}
