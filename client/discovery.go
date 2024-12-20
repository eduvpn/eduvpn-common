package client

import (
	"context"
	"sort"
	"strings"

	"codeberg.org/eduVPN/eduvpn-common/i18nerr"
	"codeberg.org/eduVPN/eduvpn-common/types/cookie"
	discotypes "codeberg.org/eduVPN/eduvpn-common/types/discovery"
)

func (c *Client) hasDiscovery() bool {
	// see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php
	return strings.HasPrefix(c.Name, "org.eduvpn.app")
}

// DiscoOrganizations gets the organizations list from the discovery server with search string `search`
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#organization-list.
func (c *Client) DiscoOrganizations(ck *cookie.Cookie, search string) (*discotypes.Organizations, error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Organization discovery with this client ID is not supported")
	}

	disco, release := c.discoMan.Discovery(true)
	defer release()

	orgs, fresh, err := disco.Organizations(ck.Context())
	if fresh {
		defer c.TrySave()
	}
	if err != nil {
		err = i18nerr.Wrap(err, "Failed to obtain the list of organizations")
	}
	if orgs == nil {
		return nil, err
	}

	// convert to public subset
	var retOrgs []discotypes.Organization
	for _, v := range orgs.List {
		if search == "" {
			retOrgs = append(retOrgs, v.Organization)
			continue
		}
		score := v.Score(search)
		if score < 0 {
			continue
		}
		v.Organization.Score = score
		retOrgs = append(retOrgs, v.Organization)
	}
	if search != "" {
		sort.Slice(retOrgs, func(i, j int) bool {
			// lower score is better
			return retOrgs[i].Score < retOrgs[j].Score
		})
	}
	return &discotypes.Organizations{
		List: retOrgs,
	}, err
}

// DiscoServers gets the servers list from the discovery server with search string `search`
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#server-list.
func (c *Client) DiscoServers(ck *cookie.Cookie, search string) (*discotypes.Servers, error) {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return nil, i18nerr.NewInternal("Server discovery with this client ID is not supported")
	}

	disco, release := c.discoMan.Discovery(true)
	defer release()
	servs, fresh, err := disco.Servers(ck.Context())
	if fresh {
		defer c.TrySave()
	}
	if err != nil {
		err = i18nerr.Wrap(err, "Failed to obtain the list of servers")
	}
	if servs == nil {
		return nil, err
	}

	// convert to public subset
	var retServs []discotypes.Server
	for _, v := range servs.List {
		if search == "" {
			retServs = append(retServs, v.Server)
			continue
		}
		score := v.Score(search)
		if score < 0 {
			continue
		}
		v.Server.Score = score
		retServs = append(retServs, v.Server)
	}
	if search != "" {
		sort.Slice(retServs, func(i, j int) bool {
			// lower score is better
			return retServs[i].Score < retServs[j].Score
		})
	}
	return &discotypes.Servers{
		List: retServs,
	}, err
}

func (c *Client) DiscoveryStartup(cb func()) error {
	// Not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return i18nerr.NewInternal("Server/organization discovery startup with this client ID is not supported")
	}

	fcb := func() {
		if cb == nil {
			return
		}

		c.mu.Lock()
		defer c.mu.Unlock()

		if c.FSM.Current != StateMain {
			return
		}

		cb()
	}
	c.discoMan.Startup(context.Background(), fcb)
	return nil
}
