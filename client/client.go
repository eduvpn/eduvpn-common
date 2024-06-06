//go:generate go run golang.org/x/text/cmd/gotext -srclang=en update -out=zgotext.go -lang=da,de,en,es,fr,it,nl,pt,sl,ukr

// Package client implements the public interface for creating eduVPN/Let's Connect! clients
package client

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/eduvpn/eduvpn-common/i18nerr"
	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/failover"
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	srvtypes "github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

// Client is the main struct for the VPN client.
type Client struct {
	// The name of the client
	Name string

	// The servers
	Servers server.Servers

	// The fsm
	FSM fsm.FSM

	// Whether to enable debugging
	Debug bool

	// TokenSetter sets the tokens in the client
	TokenSetter func(sid string, stype srvtypes.Type, tok srvtypes.Tokens)

	// TokenGetter gets the tokens from the client
	TokenGetter func(sid string, stype srvtypes.Type) *srvtypes.Tokens

	// tokenCacher
	tokCacher TokenCacher

	// cfg is the config
	cfg *config.Config

	// proxy is proxyguard
	proxy Proxy

	mu sync.Mutex
}

// MarkOrganizationsExpired marks the discovery organization list as expired
// it's a no-op if the type `t` is not secure internet
// or if discovery is nil
func (c *Client) MarkOrganizationsExpired(t srvtypes.Type) {
	if t != srvtypes.TypeSecureInternet {
		return
	}
	disco := c.cfg.Discovery()
	if disco == nil {
		return
	}
	disco.MarkOrganizationsExpired()
}

// GettingConfig is defined here to satisfy the server.Callbacks interface
// It is called when internally we are getting a config
// We go to the GettingConfig state
func (c *Client) GettingConfig() error {
	if c.FSM.InState(StateGettingConfig) {
		return nil
	}
	_, err := c.FSM.GoTransition(StateGettingConfig)
	return err
}

// InvalidProfile is defined here to satisfy the server.Callbacks interface
// It is called when a profile is invalid
// Here we call the AskProfile transition
func (c *Client) InvalidProfile(ctx context.Context, srv *server.Server) (string, error) {
	ck := cookie.NewWithContext(ctx)
	prfs, err := srv.Profiles()
	if err != nil {
		return "", err
	}
	// we are guaranteed to have profiles > 0 (even after filtering)
	// because internally this callback is only triggered if there is a choice to make

	errChan := make(chan error)
	go func() {
		err := c.FSM.GoTransitionRequired(StateAskProfile, &srvtypes.RequiredAskTransition{
			C:    ck,
			Data: prfs,
		})
		if err != nil {
			errChan <- err
		}
	}()
	pID, err := ck.Receive(errChan)
	if err != nil {
		return "", err
	}

	return pID, nil
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
	lvl := log.LevelInfo
	if debug {
		lvl = log.LevelDebug
	}

	if err = log.Logger.Init(lvl, directory); err != nil {
		return nil, i18nerr.WrapInternalf(err, "The log file with directory: '%s' failed to initialize", directory)
	}

	// set client name
	c.Name = name

	// register HTTP agent
	http.RegisterAgent(userAgentName(name), version)

	// Initialize the FSM
	c.FSM = newFSM(stateCallback, directory)

	// Debug only if given
	c.Debug = debug

	c.cfg = config.NewFromDirectory(directory)

	// set the servers
	c.Servers = server.NewServers(c.Name, c, c.cfg.V2)
	return c, nil
}

// TriggerAuth is called when authorization is triggered
// This function satisfies the server.Callbacks interface
func (c *Client) TriggerAuth(ctx context.Context, url string, wait bool) (string, error) {
	// Get a reply from the client
	if wait {
		ck := cookie.NewWithContext(ctx)
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
			return "", err
		}
		return g, nil
	}
	// Otherwise do normal authorization (desktop clients)
	err := c.FSM.GoTransitionRequired(StateOAuthStarted, url)
	if err != nil {
		return "", err
	}
	return "", nil
}

// AuthDone is called when authorization is done
// This is defined to satisfy the server.Callbacks interface
func (c *Client) AuthDone(id string, t srvtypes.Type) {
	srv, err := c.Servers.GetServer(id, t)
	if err == nil {
		srv.LastAuthorizeTime = time.Now()
	}
	_, err = c.FSM.GoTransition(StateMain)
	if err != nil {
		log.Logger.Debugf("unhandled auth done main transition: %v", err)
	}
	c.MarkOrganizationsExpired(t)
	c.TrySave()
}

// TokensUpdated is called when tokens are updated
// It updates the cache map and the client tokens
// This is defined to satisfy the server.Callbacks interface
func (c *Client) TokensUpdated(id string, t srvtypes.Type, tok eduoauth.Token) {
	if tok.Access == "" {
		return
	}
	// Set the memory
	err := c.tokCacher.Set(id, t, tok)
	if err != nil {
		log.Logger.Warningf("failed to set tokens into cache with error: %v", err)
	}

	if c.TokenSetter == nil {
		return
	}
	// Update the client
	c.TokenSetter(id, t, srvtypes.Tokens{
		Access:  tok.Access,
		Refresh: tok.Refresh,
		Expires: tok.ExpiredTimestamp.Unix(),
	})
}

// Register means updating the FSM to get to the initial state correctly
func (c *Client) Register() error {
	err := c.goTransition(StateMain)
	if err != nil {
		return err
	}
	return nil
}

// Deregister 'deregisters' the client, meaning saving the log file and the config and emptying out the client struct.
func (c *Client) Deregister() {
	// save the config
	c.TrySave()

	// Move the state machine back
	_, err := c.FSM.GoTransition(StateDeregistered)
	if err != nil {
		log.Logger.Debugf("failed deregistered transition: %v", err)
	}

	// Close the log file
	_ = log.Logger.Close()

	// Empty out the state
	*c = Client{}
}

// ExpiryTimes returns the different Unix timestamps regarding expiry
// - The time starting at which the renew button should be shown, after 30 minutes and less than 24 hours
// - The time starting at which the countdown button should be shown, less than 24 hours
// - The list of times where notifications should be shown
// These times are reset when the VPN gets disconnected
func (c *Client) ExpiryTimes() (*srvtypes.Expiry, error) {
	srv, err := c.Servers.CurrentServer()
	if err != nil {
		return nil, i18nerr.WrapInternal(err, "The current server was not found when getting the VPN expiration date")
	}
	return &srvtypes.Expiry{
		StartTime:         srv.LastAuthorizeTime.Unix(),
		EndTime:           srv.ExpireTime.Unix(),
		ButtonTime:        server.RenewButtonTime(srv.LastAuthorizeTime, srv.ExpireTime),
		CountdownTime:     server.CountdownTime(srv.LastAuthorizeTime, srv.ExpireTime),
		NotificationTimes: server.NotificationTimes(srv.LastAuthorizeTime, srv.ExpireTime),
	}, nil
}

func (c *Client) locationCallback(ck *cookie.Cookie, orgID string) error {
	locs := c.cfg.Discovery().SecureLocationList()
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
	srv, err := c.Servers.GetServer(orgID, srvtypes.TypeSecureInternet)
	if err != nil {
		return err
	}
	srv.CountryCode = loc
	c.TrySave()
	return nil
}

// TrySave tries to save the internal state file
// If an error occurs it logs it
func (c *Client) TrySave() {
	log.Logger.Debugf("saving state file")
	if c.cfg == nil {
		log.Logger.Warningf("no state file to save")
		return
	}
	err := c.cfg.Save()
	if err != nil {
		log.Logger.Warningf("failed to save state file: %v", err)
	}
}

// AddServer adds a server with identifier and type
func (c *Client) AddServer(ck *cookie.Cookie, identifier string, _type srvtypes.Type, ot *int64) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	// we are non-interactive if oauth time is non-nil
	ni := ot != nil
	// If we have failed to add the server, we remove it again
	// We add the server because we can then obtain it in other callback functions
	previousState := c.FSM.Current
	defer func() {
		// If we must run callbacks, go to the previous state if we're not in it
		if !ni && !c.FSM.InState(previousState) {
			c.FSM.GoTransition(previousState) //nolint:errcheck
		}
		if err == nil {
			c.TrySave()
		}
	}()

	if !ni {
		err = c.goTransition(StateAddingServer)
		// this is already wrapped in an UI error
		if err != nil {
			return err
		}
	}
	if _type != srvtypes.TypeSecureInternet {
		// Convert to an identifier
		identifier, err = http.EnsureValidURL(identifier, true)
		if err != nil {
			return i18nerr.WrapInternalf(err, "failed to convert identifier: %v", identifier)
		}
	}

	switch _type {
	case srvtypes.TypeInstituteAccess:
		err = c.Servers.AddInstitute(ck.Context(), c.cfg.Discovery(), identifier, ot)
		if err != nil {
			return i18nerr.Wrapf(err, "Failed to add an institute access server with URL: '%s'", identifier)
		}
	case srvtypes.TypeSecureInternet:
		err = c.Servers.AddSecure(ck.Context(), c.cfg.Discovery(), identifier, ot)
		if err != nil {
			return i18nerr.Wrapf(err, "Failed to add a secure internet server with organisation ID: '%s'", identifier)
		}
	case srvtypes.TypeCustom:
		err = c.Servers.AddCustom(ck.Context(), identifier, ot)
		if err != nil {
			return i18nerr.Wrapf(err, "Failed to add a server with URL: '%s'", identifier)
		}
	default:
		return i18nerr.NewInternalf("Failed to add server type: '%v'", _type)
	}
	return nil
}

func (c *Client) convertIdentifier(identifier string, t srvtypes.Type) (string, error) {
	// assume secure internet identifiers are always valid as we can't really assume they are valid urls (+ always https)
	if t == srvtypes.TypeSecureInternet {
		return identifier, nil
	}
	// Convert to an identifier, this also converts the scheme to HTTPS
	identifier, err := http.EnsureValidURL(identifier, true)
	if err != nil {
		return "", i18nerr.Wrapf(err, "The input: '%s' is not a valid URL", identifier)
	}
	return identifier, nil
}

// GetConfig gets a VPN configuration
func (c *Client) GetConfig(ck *cookie.Cookie, identifier string, _type srvtypes.Type, pTCP bool, startup bool) (cfg *srvtypes.Configuration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	previousState := c.FSM.Current

	defer func() {
		if err == nil {
			// it could be that we are not in getting config yet if we have just done authorization
			c.FSM.GoTransition(StateGettingConfig) //nolint:errcheck
			c.FSM.GoTransition(StateGotConfig)     //nolint:errcheck
		} else if !c.FSM.InState(previousState) {
			// go back to the previous state if an error occurred
			c.FSM.GoTransition(previousState) //nolint:errcheck
		}
		c.TrySave()
	}()

	identifier, err = c.convertIdentifier(identifier, _type)
	if err != nil {
		return nil, err
	}
	err = c.GettingConfig()
	if err != nil {
		log.Logger.Debugf("failed getting config transition: %v", err)
	}

	tok, err := c.retrieveTokens(identifier, _type)
	if err != nil {
		log.Logger.Debugf("no tokens found for server: '%s', with error: '%v'", identifier, err)
	}
	var srv *server.Server
	switch _type {
	case srvtypes.TypeInstituteAccess:
		srv, err = c.Servers.GetInstitute(ck.Context(), identifier, c.cfg.Discovery(), tok, startup)
	case srvtypes.TypeSecureInternet:
		srv, err = c.Servers.GetSecure(ck.Context(), identifier, c.cfg.Discovery(), tok, startup)

		var cErr *discovery.ErrCountryNotFound
		if errors.As(err, &cErr) {
			err = c.locationCallback(ck, identifier)
			if err == nil {
				srv, err = c.Servers.GetSecure(ck.Context(), identifier, c.cfg.Discovery(), tok, startup)
			}
		}
	case srvtypes.TypeCustom:
		srv, err = c.Servers.GetCustom(ck.Context(), identifier, tok, startup)
	default:
		err = i18nerr.NewInternalf("Server type: '%v' is not valid to get a config for", _type)
	}
	if err != nil {
		if startup {
			if errors.Is(err, api.ErrAuthorizeDisabled) {
				return nil, i18nerr.Newf("The client tried to autoconnect to the VPN server: '%s', but you need to authorizate again. Please manually connect again.", identifier)
			}
			return nil, i18nerr.Wrapf(err, "The client tried to autoconnect to the VPN server: '%s', but the operation failed to complete", identifier)
		}
		return nil, i18nerr.Wrapf(err, "Failed to connect to server: '%s'", identifier)
	}

	cfg, err = c.Servers.ConnectWithCallbacks(ck.Context(), srv, pTCP)
	if err != nil {
		return nil, i18nerr.Wrapf(err, "Failed to obtain a VPN configuration for server: '%s'", identifier)
	}
	return cfg, nil
}

// RemoveServer removes a server
func (c *Client) RemoveServer(identifier string, _type srvtypes.Type) (err error) {
	identifier, err = c.convertIdentifier(identifier, _type)
	if err != nil {
		return err
	}
	err = c.Servers.Remove(identifier, _type)
	if err != nil {
		return i18nerr.WrapInternalf(err, "Failed to remove server: '%s'", identifier)
	}
	c.MarkOrganizationsExpired(_type)
	c.TrySave()
	return nil
}

// CurrentServer gets the current server that is configured
func (c *Client) CurrentServer() (*srvtypes.Current, error) {
	curr, err := c.Servers.PublicCurrent(c.cfg.Discovery())
	if err != nil {
		return nil, i18nerr.WrapInternal(err, "The current server could not be retrieved")
	}
	return curr, nil
}

// SetProfileID set the profile ID `pID` for the current server
func (c *Client) SetProfileID(pID string) error {
	srv, err := c.Servers.CurrentServer()
	if err != nil {
		return i18nerr.WrapInternalf(err, "Failed to set the profile ID: '%s'", pID)
	}
	srv.Profiles.Current = pID
	c.TrySave()
	return nil
}

func (c *Client) retrieveTokens(sid string, t srvtypes.Type) (*eduoauth.Token, error) {
	// get from memory
	tok, err := c.tokCacher.Get(sid, t)
	if err == nil {
		return tok, nil
	}
	if c.TokenGetter == nil {
		return tok, err
	}
	// get from client
	gtok := c.TokenGetter(sid, t)
	if gtok == nil {
		return nil, errors.New("client returned nil tokens")
	}
	return &eduoauth.Token{
		Access:           gtok.Access,
		Refresh:          gtok.Refresh,
		ExpiredTimestamp: time.Unix(gtok.Expires, 0),
	}, nil
}

// Cleanup cleans up the VPN connection by sending a /disconnect
func (c *Client) Cleanup(ck *cookie.Cookie) error {
	defer c.TrySave()
	// cleanup proxyguard
	cerr := c.proxy.Cancel()
	if cerr != nil {
		log.Logger.Debugf("ProxyGuard cancel gave an error: %v", cerr)
	}
	srv, err := c.Servers.CurrentServer()
	if err != nil {
		return i18nerr.WrapInternal(err, "The current server was not found when cleaning up the connection")
	}
	tok, err := c.retrieveTokens(srv.Key.ID, srv.Key.T)
	if err != nil {
		return i18nerr.WrapInternal(err, "No OAuth tokens were found when cleaning up the connection")
	}
	auth, err := srv.ServerWithCallbacks(ck.Context(), c.cfg.Discovery(), tok, true)
	if err != nil {
		return i18nerr.WrapInternal(err, "The server was unable to be retrieved when cleaning up the connection")
	}
	err = auth.Disconnect(ck.Context())
	if err != nil {
		return i18nerr.WrapInternal(err, "Failed to cleanup the VPN connection")
	}
	return nil
}

// SetSecureLocation sets a secure internet location for
// organization ID `orgID` with country code `countryCode`
func (c *Client) SetSecureLocation(orgID string, countryCode string) error {
	// not supported with Let's Connect! & govVPN
	if !c.hasDiscovery() {
		return i18nerr.NewInternal("Setting a secure internet location with this client ID is not supported")
	}
	srv, err := c.Servers.GetServer(orgID, srvtypes.TypeSecureInternet)
	if err != nil {
		return i18nerr.WrapInternalf(err, "Failed to get the secure internet server with id: '%s' for setting a location", orgID)
	}
	srv.CountryCode = countryCode
	defer c.TrySave()

	// no cached location profiles
	if srv.LocationProfiles == nil {
		return nil
	}

	// restore profile from the location
	if v, ok := srv.LocationProfiles[srv.CountryCode]; ok {
		srv.Profiles.Current = v
	}
	return nil
}

// RenewSession is called when the user clicks on the renew session button
// It re-authorized the server by getting a server without passing tokens
func (c *Client) RenewSession(ck *cookie.Cookie) error {
	// getting the current serving with nil tokens means re-authorize
	srv, err := c.Servers.CurrentServer()
	if err != nil {
		return i18nerr.WrapInternal(err, "The current server could not be retrieved when renewing the session")
	}

	// getting a server with no tokens means re-authorize
	_, err = srv.ServerWithCallbacks(ck.Context(), c.cfg.Discovery(), nil, false)
	if err != nil {
		return i18nerr.WrapInternal(err, "The server was unable to be retrieved when renewing the session")
	}
	return nil
}

// StartFailover starts the failover procedure
func (c *Client) StartFailover(ck *cookie.Cookie, gateway string, mtu int, readRxBytes func() (int64, error)) (bool, error) {
	f := failover.New(readRxBytes)

	// get current profile
	d, err := f.Start(ck.Context(), gateway, mtu)
	if err != nil {
		return d, i18nerr.WrapInternalf(err, "Failover failed to complete with gateway: '%s' and MTU: '%d'", gateway, mtu)
	}
	return d, nil
}

// ServerList gets the list of servers
func (c *Client) ServerList() (*srvtypes.List, error) {
	g := c.cfg.V2.PublicList(c.cfg.Discovery())
	return g, nil
}
