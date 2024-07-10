package client

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	discotypes "github.com/eduvpn/eduvpn-common/types/discovery"
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
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
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
		return nil, i18nerr.NewInternal("Server/organization discovery with this client ID is not supported")
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

type DiscoManager struct {
	disco *discovery.Discovery

	cancel context.CancelFunc
	mu sync.RWMutex
	wait sync.WaitGroup
}

func (m *DiscoManager) lock(write bool) {
	if write {
		m.mu.Lock()
		return
	}
	m.mu.RLock()
}

func (m *DiscoManager) unlock(write bool) {
	if write {
		m.mu.Unlock()
		return
	}
	m.mu.RUnlock()
}

func (m *DiscoManager) Discovery(write bool) (*discovery.Discovery, func()) {
	if write {
		m.wait.Wait()
	}
	m.lock(write)
	return m.disco, func() {
		m.unlock(write)
	}
}

func (m *DiscoManager) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
	m.wait.Wait()
}

func (m *DiscoManager) Startup(ctx context.Context, cb func()) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.wait.Add(1)
	go func() {
		defer m.wait.Done()
		m.lock(false)
		discoCopy, err := m.disco.Copy()
		if err != nil {
			log.Logger.Warningf("internal error, failed to clone discovery, %v", err)
			return
		}
		m.unlock(false)
		// we already log the warning
		discoCopy.Servers(ctx) //nolint:errcheck

		m.lock(true)
		m.disco.UpdateServers(discoCopy)
		m.unlock(true)

		if cb != nil {
			cb()
		}
	}()
}
