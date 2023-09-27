//go:generate go run golang.org/x/text/cmd/gotext -srclang=en update -out=zgotext.go -lang=da,de,en,es,fr,it,nl,sl,ukr

// Package client implements the public interface for creating eduVPN/Let's Connect! clients
package client

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/internal/config"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/failover"
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	discotypes "github.com/eduvpn/eduvpn-common/types/discovery"
	srvtypes "github.com/eduvpn/eduvpn-common/types/server"
	"github.com/go-errors/errors"
)

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

func userAgentName(clientID string) string {
	switch clientID {
	case "org.eduvpn.app.windows":
		return "eduVPN for Windows"
	case "org.eduvpn.app.android":
		return "eduVPN for Android"
	case "org.eduvpn.app.ios":
		return "eduVPN for iOS"
	case "org.eduvpn.app.macos":
		return "eduVPN for macOS"
	case "org.eduvpn.app.linux":
		return "eduVPN for Linux"
	case "org.letsconnect-vpn.app.windows":
		return "Let's Connect! for Windows"
	case "org.letsconnect-vpn.app.android":
		return "Let's Connect! for Android"
	case "org.letsconnect-vpn.app.ios":
		return "Let's Connect! for iOS"
	case "org.letsconnect-vpn.app.macos":
		return "Let's Connect! for macOS"
	case "org.letsconnect-vpn.app.linux":
		return "Let's Connect! for Linux"
	default:
		return "unknown"
	}
}

func (c *Client) isLetsConnect() bool {
	// see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php
	return strings.HasPrefix(c.Name, "org.letsconnect-vpn.app")
}

// Client is the main struct for the VPN client.
type Client struct {
	// The name of the client
	Name string `json:"-"`

	// The chosen server
	Servers server.List `json:"servers"`

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

	// TokenSetter sets the tokens in the client
	TokenSetter func(srv srvtypes.Current, tok srvtypes.Tokens) `json:"-"`

	// TokenGetter gets the tokens from the client
	TokenGetter func(srv srvtypes.Current) *srvtypes.Tokens `json:"-"`

	mu sync.Mutex
}

func (c *Client) NeedsMobileRedirect() bool {
	splitted := strings.Split(c.Name, ".")
	last := splitted[len(splitted)-1]
	return last == "android" || last == "ios"
}

func (c *Client) MobileRedirect() string {
	vals := map[string]string{
		"org.letsconnect-vpn.app.ios": "org.letsconnect-vpn.app.ios:/api/callback",
		"org.letsconnect-vpn.app.android": "org.letsconnect-vpn.app:/api/callback",
		"org.eduvpn.app.ios": "org.eduvpn.app.ios:/api/callback",
		"org.eduvpn.app.android": "org.eduvpn.app:/api/callback",
	}
	return vals[c.Name]
}

func (c *Client) updateTokens(srv server.Server) error {
	if c.TokenGetter == nil {
		return errors.New("no token getter defined")
	}
	pSrv, err := c.pubCurrentServer(srv)
	if err != nil {
		return err
	}
	// shouldn't happen
	if pSrv == nil {
		return errors.New("public server is nil when getting tokens")
	}
	tokens := c.TokenGetter(*pSrv)
	if tokens == nil {
		return errors.New("client returned nil for tokens")
	}

	server.UpdateTokens(srv, oauth.Token{
		Access:           tokens.Access,
		Refresh:          tokens.Refresh,
		ExpiredTimestamp: time.Unix(tokens.Expires, 0),
	})

	return nil
}

func (c *Client) forwardTokens(srv server.Server) error {
	if c.TokenSetter == nil {
		return errors.New("no token setter defined")
	}
	pSrv, err := c.pubCurrentServer(srv)
	if err != nil {
		return err
	}
	if pSrv == nil {
		return errors.New("public server is nil when updating tokens")
	}
	o := srv.OAuth()
	if o == nil {
		return errors.New("oauth was nil when forwarding tokens")
	}
	t := o.Token()
	c.TokenSetter(*pSrv, t.Public())
	return nil
}

func (c *Client) goTransition(id fsm.StateID) error {
	handled, err := c.FSM.GoTransition(id)
	if err != nil {
		return i18nerr.WrapInternal(err, "state transition error")
	}
	if !handled {
		log.Logger.Debugf("transition not handled by the client to internal state: '%s'", GetStateName(id))
	}
	return nil
}

// New creates a new client with the following parameters:
//   - name: the name of the client
//   - directory: the directory where the config files are stored. Absolute or relative
//   - stateCallback: the callback function for the FSM that takes two states (old and new) and the data as an interface
//   - debug: whether or not we want to enable debugging
//
// It returns an error if initialization failed, for example when discovery cannot be obtained and when there are no servers.
func New(name string, version string, directory string, stateCallback func(FSMStateID, FSMStateID, interface{}) bool, debug bool) (c *Client, err error) {
	// We create the client by filling fields one by one
	c = &Client{}

	if !isAllowedClientID(name) {
		return nil, i18nerr.NewInternalf("The client registered with an invalid client ID: '%v'", name)
	}

	if len([]rune(version)) > 20 {
		return nil, i18nerr.NewInternalf("The client registered with an invalid version: '%v'", version)
	}

	// Initialize the logger
	lvl := log.LevelWarning
	if debug {
		lvl = log.LevelDebug
	}

	if err = log.Logger.Init(lvl, directory); err != nil {
		return nil, i18nerr.Wrapf(err, "The log file with directory: '%s' failed to initialize", directory)
	}

	// set client name
	c.Name = name

	// register HTTP agent
	http.RegisterAgent(userAgentName(name), version)

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

	return c, nil
}

// Registering means updating the FSM to get to the initial state correctly
func (c *Client) Register() error {
	if !c.FSM.InState(StateDeregistered) {
		return i18nerr.NewInternal("The client tried to re-initialize without deregistering first")
	}
	err := c.goTransition(StateNoServer)
	if err != nil {
		return err
	}
	return nil
}

// SaveState saves the internal state to the config
func (c *Client) SaveState() {
	log.Logger.Debugf("saving state configuration....")
	// Save the config
	if err := c.Config.Save(&c); err != nil {
		log.Logger.Infof("failed saving state configuration: '%v'", err)
	}
}

// Deregister 'deregisters' the client, meaning saving the log file and the config and emptying out the client struct.
func (c *Client) Deregister() {
	// First of all let's transition the state machine
	_ = c.goTransition(StateDeregistered)

	// SaveState saves the configuration
	c.SaveState()

	// Close the log file
	_ = log.Logger.Close()

	// Empty out the state
	*c = Client{}
}

// DiscoOrganizations gets the organizations list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#organization-list.
func (c *Client) DiscoOrganizations(ck *cookie.Cookie) (orgs *discotypes.Organizations, err error) {
	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, i18nerr.NewInternal("Server/organization discovery with Let's Connect is not supported")
	}

	// Mark organizations as expired if we have not set an organization yet
	if !c.Servers.HasSecureInternet() {
		c.Discovery.MarkOrganizationsExpired()
	}

	orgs, err = c.Discovery.Organizations(ck.Context())
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
	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, i18nerr.NewInternal("Server/organization discovery with Let's Connect is not supported")
	}

	dss, err = c.Discovery.Servers(ck.Context())
	if err != nil {
		err = i18nerr.Wrap(err, "An error occurred after getting the discovery files for the list of servers")
	}
	return
}

// ExpiryTimes returns the different Unix timestamps regarding expiry
// - The time starting at which the renew button should be shown, after 30 minutes and less than 24 hours
// - The time starting at which the countdown button should be shown, less than 24 hours
// - The list of times where notifications should be shown
// These times are reset when the VPN gets disconnected
func (c *Client) ExpiryTimes() (*srvtypes.Expiry, error) {
	// Get current expiry time
	srv, err := c.Servers.Current()
	if err != nil {
		return nil, i18nerr.Wrap(err, "The current server could not be found when getting it for expiry")
	}
	b, err := srv.Base()
	if err != nil {
		return nil, err
	}

	if b.StartTime.IsZero() {
		return nil, i18nerr.New("No start time is defined for this server")
	}

	bT := b.RenewButtonTime()
	cT := b.CountdownTime()
	nT := b.NotificationTimes()
	return &srvtypes.Expiry{
		StartTime:         b.StartTime.Unix(),
		EndTime:           b.EndTime.Unix(),
		ButtonTime:        bT,
		CountdownTime:     cT,
		NotificationTimes: nT,
	}, nil
}

func (c *Client) locationCallback(ck *cookie.Cookie) error {
	locs := c.Discovery.SecureLocationList()
	errChan := make(chan error)
	go func() {
		err := c.FSM.GoTransitionRequired(StateAskLocation, &srvtypes.RequiredAskTransition{
			C:    ck,
			Data: locs,
		})
		if err != nil {
			errChan <- err
		}
	}()
	loc, err := ck.Receive(errChan)
	if err != nil {
		return err
	}
	err = c.SetSecureLocation(ck, loc)
	if err != nil {
		return err
	}
	err = c.goTransition(StateChosenLocation)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) loginCallback(ck *cookie.Cookie, srv server.Server) error {
	// get a custom redirect
	cr := ""
	if c.NeedsMobileRedirect() {
		cr = c.MobileRedirect()
	}
	url, err := server.OAuthURL(srv, c.Name, cr)
	if err != nil {
		return err
	}
	authCodeURI := ""
	if c.NeedsMobileRedirect() {
		errChan := make(chan error)
		go func() {
			err := c.FSM.GoTransitionRequired(StateOAuthStarted, &srvtypes.RequiredAskTransition{
				C:    ck,
				Data: url,
			})
			if err != nil {
				errChan <- err
			}
		}()
		g, err := ck.Receive(errChan)
		if err != nil {
			return err
		}
		authCodeURI = g
	} else {
		err = c.FSM.GoTransitionRequired(StateOAuthStarted, url)
		if err != nil {
			return err
		}
	}
	err = server.OAuthExchange(ck.Context(), srv, authCodeURI)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) callbacks(ck *cookie.Cookie, srv server.Server, forceauth bool, startup bool) error {
	// location
	if srv.NeedsLocation() {
		if startup {
			return i18nerr.Newf("The client tried to autoconnect to the VPN server: %s, but no secure internet location is found. Please manually connect again", server.Name(srv))
		}
		err := c.locationCallback(ck)
		if err != nil {
			return i18nerr.Wrap(err, "The secure internet location could not be set")
		}
	}

	err := c.goTransition(StateChosenServer)
	if err != nil {
		log.Logger.Debugf("optional chosen server transition not possible: %v", err)
	}
	// oauth
	// TODO: This should be ck.Context()
	// But needsrelogin needs a rewrite to support this properly

	// first make sure we get the most up to date tokens from the client
	err = c.updateTokens(srv)
	if err != nil {
		log.Logger.Debugf("failed to get tokens from client: %v", err)
	}
	if server.NeedsRelogin(context.Background(), srv) || forceauth {
		if startup {
			return i18nerr.Newf("The client tried to autoconnect to the VPN server: %s, but you need to authorizate again. Please manually connect again", server.Name(srv))
		}
		// mark organizations as expired if the server is a secure internet server
		b, berr := srv.Base()
		if berr == nil && b.Type == srvtypes.TypeSecureInternet {
			c.Discovery.MarkOrganizationsExpired()
		}
		err := c.loginCallback(ck, srv)
		if err != nil {
			return i18nerr.Wrap(err, "The authorization procedure failed to complete")
		}
	}
	err = c.goTransition(StateAuthorized)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) profileCallback(ck *cookie.Cookie, srv server.Server, startup bool) error {
	vp, err := server.HasValidProfile(ck.Context(), srv, c.SupportsWireguard)
	if err != nil {
		log.Logger.Warningf("failed to determine whether the current protocol is valid with error: %v", err)
		return err
	}
	if !vp {
		if startup {
			return i18nerr.Newf("The client tried to autoconnect to the VPN server: %s, but no valid profiles were found. Please manually connect again", server.Name(srv))
		}
		vps, err := server.ValidProfiles(srv, c.SupportsWireguard)
		if err != nil {
			return i18nerr.Wrapf(err, "No suitable profiles could be found")
		}
		errChan := make(chan error)
		go func() {
			err := c.FSM.GoTransitionRequired(StateAskProfile, &srvtypes.RequiredAskTransition{
				C:    ck,
				Data: vps.Public(),
			})
			if err != nil {
				errChan <- err
			}
		}()
		pID, err := ck.Receive(errChan)
		if err != nil {
			return i18nerr.Wrapf(err, "Profile with ID: '%s' could not be set", pID)
		}
		err = server.Profile(srv, pID)
		if err != nil {
			return i18nerr.Wrapf(err, "Profile with ID: '%s' could not be obtained from the server", pID)
		}
	}
	err = c.goTransition(StateChosenProfile)
	if err != nil {
		return err
	}
	return nil
}

// AddServer adds a server with identifier and type
func (c *Client) AddServer(ck *cookie.Cookie, identifier string, _type srvtypes.Type, ni bool) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// If we have failed to add the server, we remove it again
	// We add the server because we can then obtain it in other callback functions
	previousState := c.FSM.Current
	defer func() {
		if err != nil {
			_ = c.RemoveServer(identifier, _type) //nolint:errcheck
		} else {
			c.SaveState()
		}
		// If we must run callbacks, go to the previous state if we're not in it
		if !ni && !c.FSM.InState(previousState) {
			c.FSM.GoTransition(previousState) //nolint:errcheck
		}
	}()

	if !ni {
		err = c.goTransition(StateLoadingServer)
		// this is already wrapped in an UI error
		if err != nil {
			return err
		}
	}

	if _type != srvtypes.TypeSecureInternet {
		identifier, err = http.EnsureValidURL(identifier, true)
		if err != nil {
			return i18nerr.Wrap(err, "The identifier that was passed to the library is incorrect")
		}
	}

	var srv server.Server

	switch _type {
	case srvtypes.TypeInstituteAccess:
		dSrv, err := c.Discovery.ServerByURL(identifier, "institute_access")
		if err != nil {
			return i18nerr.Wrapf(err, "Could not retrieve institute access server with URL: '%s' from discovery", identifier)
		}
		srv, err = c.Servers.AddInstituteAccess(ck.Context(), c.Name ,dSrv)
		if err != nil {
			return i18nerr.Wrapf(err, "The institute access server with URL: '%s' could not be added", identifier)
		}
	case srvtypes.TypeSecureInternet:
		dOrg, dSrv, err := c.Discovery.SecureHomeArgs(identifier)
		if err != nil {
			// We mark the organizations as expired because we got an error
			// Note that in the docs it states that it only should happen when the Org ID doesn't exist
			// However, this is nice as well because it also catches the error where the SecureInternetHome server is not found
			c.Discovery.MarkOrganizationsExpired()
			return i18nerr.Wrapf(err, "The secure internet server with organisation ID: '%s' could not be retrieved from discovery", identifier)
		}
		srv, err = c.Servers.AddSecureInternet(ck.Context(), c.Name, dOrg, dSrv)
		if err != nil {
			return i18nerr.Wrapf(err, "The secure internet server with organisation ID: '%s' could not be added", identifier)
		}
	case srvtypes.TypeCustom:
		srv, err = c.Servers.AddCustom(ck.Context(), c.Name, identifier)
		if err != nil {
			return i18nerr.Wrapf(err, "The custom server with URL: '%s' could not be added", identifier)
		}
	default:
		return i18nerr.NewInternalf("Server type: '%v' is not valid to be added", _type)
	}

	// if we are non interactive, we run no callbacks
	if ni {
		return nil
	}

	// callbacks
	err = c.callbacks(ck, srv, false, false)
	// error is already UI wrapped
	if err != nil {
		return err
	}
	terr := c.forwardTokens(srv)
	if terr != nil {
		log.Logger.Debugf("failed to forward tokens after adding: %v", terr)
	}
	return nil
}

func (c *Client) config(ck *cookie.Cookie, srv server.Server, pTCP bool, forceAuth bool, startup bool) (cfg *srvtypes.Configuration, err error) {
	// do the callbacks to ensure valid profile, location and authorization
	err = c.callbacks(ck, srv, forceAuth, startup)
	if err != nil {
		return nil, err
	}

	err = c.goTransition(StateRequestConfig)
	if err != nil {
		return nil, err
	}

	err = c.profileCallback(ck, srv, startup)
	if err != nil {
		return nil, err
	}

	cfgS, err := server.Config(ck.Context(), srv, c.SupportsWireguard, pTCP)
	if err != nil {
		return nil, i18nerr.Wrap(err, "The VPN configuration could not be obtained")
	}
	p, err := server.CurrentProfile(srv)
	if err != nil {
		return nil, i18nerr.Wrap(err, "The current profile could not be found")
	}
	pcfg := cfgS.Public(p.DefaultGateway)
	return &pcfg, nil
}

func (c *Client) server(identifier string, _type srvtypes.Type) (srv server.Server, setter func(server.Server) error, err error) {
	switch _type {
	case srvtypes.TypeInstituteAccess:
		srv, err = c.Servers.InstituteAccess(identifier)
		setter = c.Servers.SetInstituteAccess
	case srvtypes.TypeSecureInternet:
		srv, err = c.Servers.SecureInternet(identifier)
		setter = c.Servers.SetSecureInternet
	case srvtypes.TypeCustom:
		srv, err = c.Servers.CustomServer(identifier)
		setter = c.Servers.SetCustom
	default:
		return nil, nil, i18nerr.NewInternalf("Not a valid server type: %v", _type)
	}
	return srv, setter, err
}

// GetConfig gets a VPN configuration
func (c *Client) GetConfig(ck *cookie.Cookie, identifier string, _type srvtypes.Type, pTCP bool, startup bool) (cfg *srvtypes.Configuration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	previousState := c.FSM.Current
	defer func() {
		if err == nil {
			c.FSM.GoTransition(StateGotConfig) //nolint:errcheck
			c.SaveState()
		} else if !c.FSM.InState(previousState) {
			// go back to the previous state if an error occurred
			c.FSM.GoTransition(previousState) //nolint:errcheck
		}
	}()
	if _type != srvtypes.TypeSecureInternet {
		identifier, err = http.EnsureValidURL(identifier, true)
		if err != nil {
			return nil, i18nerr.Wrapf(err, "Identifier: '%s' for server with type: '%d' is not valid", identifier, _type)
		}
	}
	err = c.goTransition(StateLoadingServer)
	if err != nil {
		return nil, err
	}
	srv, set, err := c.server(identifier, _type)
	if err != nil {
		return nil, err
	}
	// refresh the server endpoints
	err = srv.RefreshEndpoints(ck.Context(), &c.Discovery)

	// If we get a canceled error, return that, otherwise just log the error
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, i18nerr.Wrap(err, "The operation for getting a VPN configuration was canceled")
		}

		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}

	// get a config and retry with authorization if expired
	cfg, err = c.config(ck, srv, pTCP, false, startup)
	tErr := &oauth.TokensInvalidError{}
	if err != nil && errors.As(err, &tErr) {
		log.Logger.Debugf("the tokens were invalid, trying again...")
		cfg, err = c.config(ck, srv, pTCP, true, startup)
	}

	// tokens might be updated, forward them
	defer func() {
		terr := c.forwardTokens(srv)
		if terr != nil {
			log.Logger.Debugf("failed to forward tokens after get config: %v", terr)
		}
	}()

	// still an error, return nil with the error
	if err != nil {
		return nil, err
	}

	// set the current server
	if err = set(srv); err != nil {
		return nil, i18nerr.Wrapf(err, "Failed to set the server with identifier: '%s' as the current", identifier)
	}

	return cfg, nil
}

func (c *Client) RemoveServer(identifier string, _type srvtypes.Type) (err error) {
	if _type != srvtypes.TypeSecureInternet {
		identifier, err = http.EnsureValidURL(identifier, true)
		if err != nil {
			return i18nerr.Wrapf(err, "Identifier: '%s' for server with type: '%d' is not valid for removal", identifier, _type)
		}
	}
	// miscellaneous error
	var mErr error
	switch _type {
	case srvtypes.TypeInstituteAccess:
		mErr = c.Servers.RemoveInstituteAccess(identifier)
	case srvtypes.TypeSecureInternet:
		mErr = c.Servers.RemoveSecureInternet(identifier)
	case srvtypes.TypeCustom:
		mErr = c.Servers.RemoveCustom(identifier)
	default:
		return i18nerr.NewInternalf("Not a valid server type: %v", _type)
	}
	if mErr != nil {
		log.Logger.Debugf("failed to remove server with identifier: '%s' and type: '%d', error: %v", identifier, _type, mErr)
	}
	c.SaveState()
	return nil
}

func (c *Client) CurrentServer() (*srvtypes.Current, error) {
	srv, err := c.Servers.Current()
	if err != nil {
		return nil, err
	}
	return c.pubCurrentServer(srv)
}

func (c *Client) pubCurrentServer(srv server.Server) (*srvtypes.Current, error) {
	b, err := srv.Base()
	if err != nil {
		return nil, err
	}
	pub, err := srv.Public()
	if err != nil {
		return nil, err
	}
	switch t := pub.(type) {
	case *srvtypes.Server:
		if b.Type == srvtypes.TypeInstituteAccess {
			return &srvtypes.Current{
				Institute: &srvtypes.Institute{
					Server:          *t,
					SupportContacts: b.SupportContact,
					// TODO: delisted
					Delisted: false,
				},
				Type: srvtypes.TypeInstituteAccess,
			}, nil
		}
		return &srvtypes.Current{
			Custom: t,
			Type:   srvtypes.TypeCustom,
		}, nil
	case *srvtypes.SecureInternet:
		t.SupportContacts = b.SupportContact
		t.Locations = c.Discovery.SecureLocationList()
		return &srvtypes.Current{
			SecureInternet: t,
			Type:           srvtypes.TypeSecureInternet,
		}, nil
	default:
		panic("unknown type")
	}
}

// TODO: This should not rely on interface{}
func (c *Client) pubServer(srv server.Server) (interface{}, error) {
	pub, err := srv.Public()
	if err != nil {
		return nil, err
	}
	b, err := srv.Base()
	if err != nil {
		return nil, err
	}
	switch t := pub.(type) {
	case *srvtypes.Server:
		if b.Type == srvtypes.TypeInstituteAccess {
			return &srvtypes.Institute{
				Server:          *t,
				SupportContacts: b.SupportContact,
				// TODO: delisted
				Delisted: false,
			}, nil
		}
		return t, nil
	case *srvtypes.SecureInternet:
		t.SupportContacts = b.SupportContact
		t.Locations = c.Discovery.SecureLocationList()
		return t, nil
	default:
		panic("unknown type")
	}
}

func (c *Client) ServerList() (*srvtypes.List, error) {
	if c.FSM.InState(StateDeregistered) {
		return nil, i18nerr.NewInternal("Client is not registered")
	}
	var customServers []srvtypes.Server
	for _, v := range c.Servers.CustomServers.Map {
		if v == nil {
			continue
		}
		p, err := c.pubServer(v)
		if err != nil {
			continue
		}
		c, ok := p.(*srvtypes.Server)
		if !ok {
			continue
		}
		customServers = append(customServers, *c)
	}
	var instituteServers []srvtypes.Institute
	for _, v := range c.Servers.InstituteServers.Map {
		if v == nil {
			continue
		}
		p, err := c.pubServer(v)
		if err != nil {
			continue
		}
		i, ok := p.(*srvtypes.Institute)
		if !ok {
			continue
		}
		instituteServers = append(instituteServers, *i)
	}
	var secureInternet *srvtypes.SecureInternet
	if c.Servers.HasSecureInternet() {
		srv := &c.Servers.SecureInternetHomeServer
		p, err := c.pubServer(srv)
		if err == nil {
			s, ok := p.(*srvtypes.SecureInternet)
			if ok {
				secureInternet = s
			}
		}
	}
	return &srvtypes.List{
		Institutes:     instituteServers,
		SecureInternet: secureInternet,
		Custom:         customServers,
	}, nil
}

func (c *Client) SetProfileID(pID string) (err error) {
	srv, err := c.Servers.Current()
	if err != nil {
		return err
	}
	err = server.Profile(srv, pID)
	if err == nil {
		c.SaveState()
	}
	return err
}

func (c *Client) Cleanup(ck *cookie.Cookie) (err error) {
	// get the current server
	srv, err := c.Servers.Current()
	if err != nil {
		return i18nerr.Wrap(err, "Failed to get the current server to cleanup the connection")
	}

	err = srv.RefreshEndpoints(ck.Context(), &c.Discovery)

	// If we get a canceled error, return that, otherwise just log the error
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return i18nerr.Wrap(err, "The cleanup process was canceled")
		}

		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}


	defer c.SaveState()
	err = c.updateTokens(srv)
	if err != nil {
		log.Logger.Debugf("failed to update tokens for disconnect: %v", err)
	}
	err = server.Disconnect(ck.Context(), srv)
	if err != nil {
		return i18nerr.Wrap(err, "Failed to cleanup the VPN connection for the current server")
	}
	err = c.forwardTokens(srv)
	if err != nil {
		log.Logger.Debugf("failed to forward tokens after disconnect: %v", err)
	}
	return nil
}

func (c *Client) SetSecureLocation(ck *cookie.Cookie, countryCode string) (err error) {
	if c.isLetsConnect() {
		return i18nerr.NewInternal("Setting a secure internet location with Let's Connect! is not supported")
	}

	if !c.Servers.HasSecureInternet() {
		return i18nerr.Newf("No secure internet server available to set a location for")
	}

	dSrv, err := c.Discovery.ServerByCountryCode(countryCode)
	if err != nil {
		return err
	}

	err = c.Servers.SecureInternetHomeServer.Location(ck.Context(), dSrv)
	if err == nil {
		c.SaveState()
	}
	return err
}

func (c *Client) RenewSession(ck *cookie.Cookie) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	srv, err := c.Servers.Current()
	if err != nil {
		return i18nerr.Wrap(err, "Failed to get current server for renewing the session")
	}
	// The server has not been chosen yet, this means that we want to manually renew
	// TODO: is this needed?
	if !c.FSM.InState(StateLoadingServer) {
		c.FSM.GoTransition(StateLoadingServer) //nolint:errcheck
	}
	err = srv.RefreshEndpoints(ck.Context(), &c.Discovery)

	// If we get a canceled error, return that, otherwise just log the error
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return i18nerr.Wrap(err, "The renewing process was canceled")
		}

		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}


	// update tokens in the end
	defer func() {
		terr := c.forwardTokens(srv)
		if terr != nil {
			log.Logger.Debugf("failed to forward tokens after renew: %v", terr)
		}
	}()
	defer c.SaveState()
	// TODO: Maybe this can be deleted because we force auth now
	server.MarkTokensForRenew(srv)
	// run the callbacks by forcing auth
	return c.callbacks(ck, srv, true, false)
}

func (c *Client) StartFailover(ck *cookie.Cookie, gateway string, mtu int, readRxBytes func() (int64, error)) (bool, error) {
	f := failover.New(readRxBytes)

	d, err := f.Start(ck.Context(), gateway, mtu)
	if err != nil {
		return d, i18nerr.Wrapf(err, "Failover failed to complete with gateway: '%s' and MTU: '%d'", gateway, mtu)
	}
	return d, nil
}

func (c *Client) SetState(state FSMStateID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	curr := c.FSM.Current
	_, err := c.FSM.GoTransition(state)
	if err != nil {
		// self-transitions are only debug errors
		if c.FSM.InState(state) {
			log.Logger.Debugf("attempt an invalid self-transition: %s", c.FSM.GetStateName(state))
			return nil
		}
		return i18nerr.WrapInternalf(err, "Failed internal state transition requested by the client from: '%s' to '%s'", GetStateName(curr), GetStateName(state))
	}
	return nil
}

func (c *Client) InState(state FSMStateID) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.FSM.InState(state)
}
