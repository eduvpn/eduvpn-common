// Package client implements the public interface for creating eduVPN/Let's Connect! clients
package client

import (
	"fmt"
	"strings"

	"github.com/eduvpn/eduvpn-common/internal/config"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/failover"
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
	"github.com/go-errors/errors"
)

type (
	// ServerBase is an alias to the internal ServerBase
	// This contains the details for each server.
	ServerBase = server.Base
)

func (c *Client) logError(err error) {
	// Logs the error with the same level/verbosity as the error
	if c.Debug {
		log.Logger.Inherit(err, fmt.Sprintf("\nwith stacktrace: %s\n", err.(*errors.Error).ErrorStack()))
	} else {
		log.Logger.Inherit(err, "")
	}
}

func (c *Client) isLetsConnect() bool {
	// see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php
	return strings.HasPrefix(c.Name, "org.letsconnect-vpn.app")
}

// isAllowedClientID checks if the 'clientID' is in the list of allowed client IDs
func isAllowedClientID(clientID string) bool {
	allowList := []string{
		// eduVPN
		"org.eduvpn.app.windows",
		"org.eduvpn.app.android",
		"org.eduvpn.app.ios",
		"org.eduvpn.app.macos",
		"org.eduvpn.app.linux",
		// Let's Connect!
		"org.letsconnect-vpn.app.windows",
		"org.letsconnect-vpn.app.android",
		"org.letsconnect-vpn.app.ios",
		"org.letsconnect-vpn.app.macos",
		"org.letsconnect-vpn.app.linux",
	}
	for _, x := range allowList {
		if x == clientID {
			return true
		}
	}
	return false
}

// Client is the main struct for the VPN client.
type Client struct {
	// The name of the client
	Name string `json:"-"`

	// The language used for language matching
	Language string `json:"-"` // language should not be saved

	// The chosen server
	Servers server.Servers `json:"servers"`

	// The list of servers and organizations from disco
	Discovery discovery.Discovery `json:"discovery"`

	// The fsm
	FSM fsm.FSM `json:"-"`

	// The config
	Config config.Config `json:"-"`

	// Whether or not this client supports WireGuard
	SupportsWireguard bool `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`

	// The Failover monitor for the current VPN connection
	Failover *failover.DroppedConMon `json:"-"`
}

// Register initializes the clientwith the following parameters:
//   - name: the name of the client
//   - directory: the directory where the config files are stored. Absolute or relative
//   - stateCallback: the callback function for the FSM that takes two states (old and new) and the data as an interface
//   - debug: whether or not we want to enable debugging
//
// It returns an error if initialization failed, for example when discovery cannot be obtained and when there are no servers.
func (c *Client) Register(
	name string,
	directory string,
	language string,
	stateCallback func(FSMStateID, FSMStateID, interface{}) bool,
	debug bool,
) (err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	if !c.InFSMState(StateDeregistered) {
		return errors.Errorf("fsm attempt to register while in '%v'", c.FSM.Current)
	}

	if !isAllowedClientID(name) {
		return errors.Errorf("client ID is not allowed: '%v', see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php for a list of allowed IDs", name)
	}

	c.Name = name

	// TODO: Verify language setting?
	c.Language = language

	// Initialize the logger
	lvl := log.LevelWarning
	if debug {
		lvl = log.LevelDebug
	}

	if err = log.Logger.Init(lvl, directory); err != nil {
		return err
	}

	// Initialize the FSM
	c.FSM = newFSM(stateCallback, directory, debug)

	// By default we support wireguard
	c.SupportsWireguard = true

	// Debug only if given
	c.Debug = debug

	// Initialize the Config
	c.Config.Init(directory, "state")

	// Try to load the previous configuration
	if c.Config.Load(&c) != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		log.Logger.Infof("Previous configuration not found")
	}

	// Go to the No Server state with the saved servers after we're done
	defer c.FSM.GoTransitionWithData(StateNoServer, c.Servers)

	// Let's Connect! doesn't care about discovery
	if c.isLetsConnect() {
		return nil
	}

	// Check if we are able to fetch discovery, and log if something went wrong
	if _, err := c.DiscoServers(); err != nil {
		log.Logger.Warningf("Failed to get discovery servers: %v", err)
	}

	if _, err := c.DiscoOrganizations(); err != nil {
		log.Logger.Warningf("Failed to get discovery organizations: %v", err)
	}

	return nil
}

// Deregister 'deregisters' the client, meaning saving the log file and the config and emptying out the client struct.
func (c *Client) Deregister() {
	// Close the log file
	_ = log.Logger.Close()

	// Save the config
	if err := c.Config.Save(&c); err != nil {
		log.Logger.Infof("c.Config.Save failed: %s\nstacktrace:\n%s", err.Error(), err.(*errors.Error).ErrorStack())
	}

	// Empty out the state
	*c = Client{}
}

// askProfile asks the user for a profile by moving the FSM to the ASK_PROFILE state.
func (c *Client) askProfile(srv server.Server) error {
	ps, err := server.ValidProfiles(srv, c.SupportsWireguard)
	if err != nil {
		return err
	}
	if err = c.FSM.GoTransitionRequired(StateAskProfile, ps); err != nil {
		return err
	}
	return nil
}

// DiscoOrganizations gets the organizations list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#organization-list.
func (c *Client) DiscoOrganizations() (orgs *types.DiscoveryOrganizations, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, errors.Errorf("discovery with Let's Connect is not supported")
	}

	// Mark organizations as expired if we have not set an organization yet
	if !c.Servers.HasSecureInternet() {
		c.Discovery.MarkOrganizationsExpired()
	}

	return c.Discovery.Organizations()
}

// DiscoServers gets the servers list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#server-list.
func (c *Client) DiscoServers() (dss *types.DiscoveryServers, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, errors.Errorf("discovery with Let's Connect is not supported")
	}

	return c.Discovery.Servers()
}

// GetTranslated gets the translation for `languages` using the current state language.
func (c *Client) GetTranslated(languages map[string]string) string {
	return util.GetLanguageMatched(languages, c.Language)
}
