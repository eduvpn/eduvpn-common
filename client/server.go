package client

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/failover"
	"github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/types"
	"github.com/go-errors/errors"
)

type ConfigData = server.ConfigData

// getConfigAuth gets a config with authorization and authentication.
// It also asks for a profile if no valid profile is found.
func (c *Client) getConfigAuth(srv server.Server, preferTCP bool, t oauth.Token) (*ConfigData, error) {
	err := c.ensureLogin(srv, t)
	if err != nil {
		return nil, err
	}

	// TODO(jwijenbergh): Should we check if it returns false?
	c.FSM.GoTransition(StateRequestConfig)

	ok, err := server.HasValidProfile(srv, c.SupportsWireguard)
	if err != nil {
		return nil, err
	}

	// No valid profile, ask for one
	if !ok {
		if err = c.askProfile(srv); err != nil {
			return nil, err
		}
	}

	// We return the error otherwise we wrap it too much
	return server.Config(srv, c.SupportsWireguard, preferTCP)
}

// retryConfigAuth retries the getConfigAuth function if the tokens are invalid.
// If OAuth is cancelled, it makes sure that we only forward the error as additional info.
func (c *Client) retryConfigAuth(srv server.Server, preferTCP bool, t oauth.Token) (*ConfigData, error) {
	cfg, err := c.getConfigAuth(srv, preferTCP, t)
	if err == nil {
		return cfg, nil
	}
	// Only retry if the error is that the tokens are invalid
	tErr := &oauth.TokensInvalidError{}
	if errors.As(err, &tErr) {
		// TODO: Is passing empty tokens correct here?
		cfg, err = c.getConfigAuth(srv, preferTCP, oauth.Token{})
		if err == nil {
			return cfg, nil
		}
	}
	c.goBackInternal()
	return nil, err
}

// getConfig gets an OpenVPN/WireGuard configuration by contacting the server, moving the FSM towards the DISCONNECTED state and then saving the local configuration file.
func (c *Client) getConfig(srv server.Server, preferTCP bool, t oauth.Token) (*ConfigData, error) {
	if c.InFSMState(StateDeregistered) {
		return nil, errors.Errorf("getConfig attempt in '%v'", StateDeregistered)
	}

	// Refresh the server endpoints
	// This is the best effort
	err := srv.RefreshEndpoints(&c.Discovery)
	if err != nil {
		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}

	cfg, err := c.retryConfigAuth(srv, preferTCP, t)
	if err != nil {
		return nil, err
	}

	srv1, err := c.Servers.GetCurrentServer()
	if err != nil {
		return nil, err
	}

	// Signal the server display info
	c.FSM.GoTransitionWithData(StateDisconnected, srv1)

	// Save the config
	if err = c.Config.Save(&c); err != nil {
		// TODO(jwijenbergh): Not sure why INFO level, yet stacktrace...
		// TODO(jwijenbergh): Even worse, why logging it but then return nil? The calling code will think that everything went well.
		log.Logger.Infof("c.Config.Save failed: %s\nstacktrace:\n%s",
			err.Error(), err.(*errors.Error).ErrorStack())
	}

	return cfg, nil
}

// Cleanup cleans up the VPN connection by sending a /disconnect to the server
func (c *Client) Cleanup(ct oauth.Token) error {
	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		c.logError(err)
		return err
	}
	err = srv.RefreshEndpoints(&c.Discovery)
	if err != nil {
		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}

	// If we need to relogin, update tokens
	if server.NeedsRelogin(srv) {
		server.UpdateTokens(srv, ct)
	}
	// update tokens to client
	defer c.ForwardTokenUpdate(srv)
	// Do the /disconnect API call
	err = server.Disconnect(srv)
	if err != nil {
		// We log nothing here because this can happen regularly
		// Maybe we should not log errors that we return directly anyways?
		return err
	}
	// TODO: Tokens might be refreshed, return updated tokens
	// Not implemented yet, because ideally we want this implemented with an interface
	return nil
}

// SetSecureLocation sets the location for the current secure location server. countryCode is the secure location to be chosen.
// This function returns an error e.g. if the server cannot be found or the location is wrong.
func (c *Client) SetSecureLocation(countryCode string) error {
	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		err := errors.Errorf("discovery with Let's Connect is not supported")
		c.logError(err)
		return err
	}

	srv, err := c.Discovery.ServerByCountryCode(countryCode)
	if err != nil {
		c.goBackInternal()
		c.logError(err)
		return err
	}

	if err = c.Servers.SetSecureLocation(srv); err != nil {
		c.goBackInternal()
		c.logError(err)
	}
	return err
}

// RemoveSecureInternet removes the current secure internet server.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (c *Client) RemoveSecureInternet() error {
	if c.InFSMState(StateDeregistered) {
		err := errors.Errorf("RemoveSecureInternet attempt in '%v'", StateDeregistered)
		c.logError(err)
		return err
	}
	// No error because we can only have one secure internet server and if there are no secure internet servers, this is a NO-OP
	c.Servers.RemoveSecureInternet()
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)
	// Save the config
	if err := c.Config.Save(&c); err != nil {
		// TODO(jwijenbergh): Not sure why INFO level, yet stacktrace...
		// TODO(jwijenbergh): Even worse, why logging it but then return nil? The calling code will think that everything went well.
		log.Logger.Infof("c.Config.Save failed: %s\nstacktrace:\n%s",
			err.Error(), err.(*errors.Error).ErrorStack())
	}
	return nil
}

// RemoveInstituteAccess removes the institute access server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (c *Client) RemoveInstituteAccess(url string) error {
	if c.InFSMState(StateDeregistered) {
		err := errors.Errorf("RemoveInstituteAccess attempt in '%v'", StateDeregistered)
		c.logError(err)
		return err
	}
	// No error because this is a NO-OP if the server doesn't exist
	c.Servers.RemoveInstituteAccess(url)
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)
	// Save the config
	if err := c.Config.Save(&c); err != nil {
		// TODO(jwijenbergh): Not sure why INFO level, yet stacktrace...
		// TODO(jwijenbergh): Even worse, why logging it but then return nil? The calling code will think that everything went well.
		log.Logger.Infof("c.Config.Save failed: %s\nstacktrace:\n%s",
			err.Error(), err.(*errors.Error).ErrorStack())
	}
	return nil
}

// RemoveCustomServer removes the custom server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (c *Client) RemoveCustomServer(url string) error {
	if c.InFSMState(StateDeregistered) {
		err := errors.Errorf("RemoveCustomServer attempt in '%v'", StateDeregistered)
		c.logError(err)
		return err
	}
	// No error because this is a NO-OP if the server doesn't exist
	c.Servers.RemoveCustomServer(url)
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)
	// Save the config
	if err := c.Config.Save(&c); err != nil {
		// TODO(jwijenbergh): Not sure why INFO level, yet stacktrace...
		// TODO(jwijenbergh): Even worse, why logging it but then return nil? The calling code will think that everything went well.
		log.Logger.Infof("c.Config.Save failed: %s\nstacktrace:\n%s",
			err.Error(), err.(*errors.Error).ErrorStack())
	}
	return nil
}

// AddInstituteServer adds an Institute Access server by `url`.
func (c *Client) AddInstituteServer(url string) (srv server.Server, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		err = errors.Errorf("discovery with Let's Connect is not supported")
		return nil, err
	}

	// Indicate that we're loading the server
	c.FSM.GoTransition(StateLoadingServer)

	// FIXME: Do nothing with discovery here as the client already has it
	// So pass a server as the parameter
	var dSrv *types.DiscoveryServer
	dSrv, err = c.Discovery.ServerByURL(url, "institute_access")
	if err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Add the secure internet server
	srv, err = c.Servers.AddInstituteAccessServer(dSrv)
	if err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current so OAuth can be cancelled
	if err = c.Servers.SetInstituteAccess(srv); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Indicate that we want to authorize this server
	c.FSM.GoTransition(StateChosenServer)

	// Authorize it
	if err = c.ensureLogin(srv, oauth.Token{}); err != nil {
		// Removing is best effort
		_ = c.RemoveInstituteAccess(url)
		return nil, err
	}

	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)
	return srv, nil
}

// AddSecureInternetHomeServer adds a Secure Internet Home Server with `orgID` that was obtained from the Discovery file.
// Because there is only one Secure Internet Home Server, it replaces the existing one.
func (c *Client) AddSecureInternetHomeServer(orgID string) (srv server.Server, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, errors.Errorf("discovery with Let's Connect is not supported")
	}

	// Indicate that we're loading the server
	c.FSM.GoTransition(StateLoadingServer)

	// Get the secure internet URL from discovery
	org, dSrv, err := c.Discovery.SecureHomeArgs(orgID)
	if err != nil {
		// We mark the organizations as expired because we got an error
		// Note that in the docs it states that it only should happen when the Org ID doesn't exist
		// However, this is nice as well because it also catches the error where the SecureInternetHome server is not found
		c.Discovery.MarkOrganizationsExpired()
		c.goBackInternal()
		return nil, err
	}

	// Add the secure internet server
	srv, err = c.Servers.AddSecureInternet(org, dSrv)
	if err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current so OAuth can be cancelled
	if err = c.Servers.SetSecureInternet(srv); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Server has been chosen for authentication
	c.FSM.GoTransition(StateChosenServer)

	// Authorize it
	if err = c.ensureLogin(srv, oauth.Token{}); err != nil {
		// Removing is best effort
		_ = c.RemoveSecureInternet()
		return nil, err
	}
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)
	return srv, nil
}

// AddCustomServer adds a Custom Server by `url`.
func (c *Client) AddCustomServer(url string) (srv server.Server, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	if url, err = http.EnsureValidURL(url); err != nil {
		return nil, err
	}

	// Indicate that we're loading the server
	c.FSM.GoTransition(StateLoadingServer)

	customServer := &types.DiscoveryServer{
		BaseURL:     url,
		DisplayName: map[string]string{"en": url},
		Type:        "custom_server",
	}

	// A custom server is just an institute access server under the hood
	if srv, err = c.Servers.AddCustomServer(customServer); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current so OAuth can be cancelled
	if err = c.Servers.SetCustomServer(srv); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Server has been chosen for authentication
	c.FSM.GoTransition(StateChosenServer)

	// Authorize it
	if err = c.ensureLogin(srv, oauth.Token{}); err != nil {
		// removing is best effort
		_ = c.RemoveCustomServer(url)
		return nil, err
	}

	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)
	return srv, nil
}

// GetConfigInstituteAccess gets a configuration for an Institute Access Server.
// It ensures that the Institute Access Server exists by creating or using an existing one with the url.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (c *Client) GetConfigInstituteAccess(url string, preferTCP bool, t oauth.Token) (cfg *ConfigData, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, errors.Errorf("discovery with Let's Connect is not supported")
	}

	c.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	var srv *server.InstituteAccessServer
	if srv, err = c.Servers.GetInstituteAccess(url); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current
	if err = c.Servers.SetInstituteAccess(srv); err != nil {
		return nil, err
	}

	// The server has now been chosen
	c.FSM.GoTransition(StateChosenServer)

	if cfg, err = c.getConfig(srv, preferTCP, t); err != nil {
		c.goBackInternal()
	}

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)

	return cfg, err
}

// GetConfigSecureInternet gets a configuration for a Secure Internet Server.
// It ensures that the Secure Internet Server exists by creating or using an existing one with the orgID.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (c *Client) GetConfigSecureInternet(orgID string, preferTCP bool, t oauth.Token) (cfg *ConfigData, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	log.Logger.Debugf("getting config for secure internet server with org ID: '%s", orgID)

	// Not supported with Let's Connect!
	if c.isLetsConnect() {
		return nil, errors.Errorf("discovery with Let's Connect is not supported")
	}

	c.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	var srv *server.SecureInternetHomeServer
	if srv, err = c.Servers.GetSecureInternetHomeServer(); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current
	if err = c.Servers.SetSecureInternet(srv); err != nil {
		return nil, err
	}

	c.FSM.GoTransition(StateChosenServer)

	if cfg, err = c.getConfig(srv, preferTCP, t); err != nil {
		c.goBackInternal()
	}

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)

	return cfg, err
}

// GetConfigCustomServer gets a configuration for a Custom Server.
// It ensures that the Custom Server exists by creating or using an existing one with the url.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (c *Client) GetConfigCustomServer(url string, preferTCP bool, t oauth.Token) (cfg *ConfigData, err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	if url, err = http.EnsureValidURL(url); err != nil {
		return nil, err
	}

	c.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	var srv *server.InstituteAccessServer
	if srv, err = c.Servers.GetCustomServer(url); err != nil {
		c.goBackInternal()
		return nil, err
	}

	// Set the server as the current
	if err = c.Servers.SetCustomServer(srv); err != nil {
		c.goBackInternal()
		return nil, err
	}

	c.FSM.GoTransition(StateChosenServer)

	if cfg, err = c.getConfig(srv, preferTCP, t); err != nil {
		c.goBackInternal()
	}

	// Also forward tokens using the callback
	c.ForwardTokenUpdate(srv)

	return cfg, err
}

// askSecureLocation asks the user to choose a Secure Internet location by moving the FSM to the STATE_ASK_LOCATION state.
func (c *Client) askSecureLocation() error {
	loc := c.Discovery.SecureLocationList()

	// Ask for the location in the callback
	if err := c.FSM.GoTransitionRequired(StateAskLocation, loc); err != nil {
		return err
	}

	// The state has changed, meaning setting the secure location was not successful
	if c.FSM.Current != StateAskLocation {
		log.Logger.Debugf("fsm failed to transit; expected %v / actual %v", GetStateName(StateAskLocation), GetStateName(c.FSM.Current))
		return errors.New("failed loading secure internet location")
	}
	return nil
}

// ChangeSecureLocation changes the location for an existing Secure Internet Server.
// Changing a secure internet location is only possible when the user is in the main screen (STATE_NO_SERVER), otherwise it returns an error.
// It also returns an error if something has gone wrong when selecting the new location.
func (c *Client) ChangeSecureLocation() error {
	if !c.InFSMState(StateNoServer) {
		return errors.Errorf("ChangeSecureLocation attempt in %v (only %v allowed)",
			c.FSM.Current, StateNoServer)
	}

	if err := c.askSecureLocation(); err != nil {
		c.logError(err)
		return err
	}

	// Go back to the main screen
	c.FSM.GoTransitionWithData(StateNoServer, c.Servers)

	return nil
}

// RenewSession renews the session for the current VPN server.
// This logs the user back in.
func (c *Client) RenewSession() (err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	var srv server.Server
	if srv, err = c.Servers.GetCurrentServer(); err != nil {
		return err
	}

	err = srv.RefreshEndpoints(&c.Discovery)
	if err != nil {
		log.Logger.Warningf("failed to refresh server endpoints: %v", err)
	}

	// The server has not been chosen yet, this means that we want to manually renew
	if c.FSM.InState(StateNoServer) {
		c.FSM.GoTransition(StateChosenServer)
	}

	server.MarkTokensForRenew(srv)
	err = c.ensureLogin(srv, oauth.Token{})
	c.ForwardTokenUpdate(srv)
	return err
}

// ShouldRenewButton returns true if the renew button should be shown
// If there is no server then this returns false and logs with INFO if so
// In other cases it simply checks the expiry time and calculates according to: https://github.com/eduvpn/documentation/blob/b93854dcdd22050d5f23e401619e0165cb8bc591/API.md#session-expiry.
func (c *Client) ShouldRenewButton() bool {
	if !c.InFSMState(StateConnected) && !c.InFSMState(StateConnecting) &&
		!c.InFSMState(StateDisconnected) &&
		!c.InFSMState(StateDisconnecting) {
		return false
	}

	srv, err := c.Servers.GetCurrentServer()
	if err != nil {
		log.Logger.Infof("no server to renew: %s\nstacktrace:\n%s", err.Error(), err.(*errors.Error).ErrorStack())
		return false
	}

	return server.ShouldRenewButton(srv)
}

// ensureLogin logs the user back in if needed.
// It runs the FSM transitions to ask for user input.
func (c *Client) ensureLogin(srv server.Server, ct oauth.Token) (err error) {
	// Relogin with oauth
	// This moves the state to authorized
	if !server.NeedsRelogin(srv) {
		// OAuth was valid, ensure we are in the authorized state
		c.FSM.GoTransition(StateAuthorized)
		return nil
	}

	// Try again but update the tokens using the client provided tokens
	server.UpdateTokens(srv, ct)
	if !server.NeedsRelogin(srv) {
		// OAuth was valid, ensure we are in the authorized state
		c.FSM.GoTransition(StateAuthorized)
		return nil
	}

	// Mark organizations as expired if the server is a secure internet server
	b, err := srv.Base()
	// We only try to update it when we found the server base
	if err == nil && b.Type == "secure_internet" {
		c.Discovery.MarkOrganizationsExpired()
	}

	// Tokens are not valid or the client gave an error when updating tokens
	// Otherwise, do the OAuth exchange
	var url string
	if url, err = server.OAuthURL(srv, c.Name); err != nil {
		return err
	}

	if err = c.FSM.GoTransitionRequired(StateOAuthStarted, url); err != nil {
		return err
	}

	if err = server.OAuthExchange(srv); err != nil {
		c.goBackInternal()
	}
	c.FSM.GoTransition(StateAuthorized)
	b, berr := srv.Base()
	if berr == nil {
		b.StartTimeOAuth = time.Now()
	}

	return err
}

// SetProfileID sets a `profileID` for the current server.
// An error is returned if this is not possible, for example when no server is configured.
func (c *Client) SetProfileID(profileID string) (err error) {
	defer func() {
		if err != nil {
			c.logError(err)
		}
	}()

	var srv server.Server
	if srv, err = c.Servers.GetCurrentServer(); err != nil {
		c.goBackInternal()
		return err
	}

	var b *server.Base
	if b, err = srv.Base(); err != nil {
		c.goBackInternal()
		return err
	}
	b.Profiles.Current = profileID
	return nil
}

func (c *Client) StartFailover(gateway string, wgMTU int, readRxBytes func() (int64, error)) (bool, error) {
	currentServer, currentServerErr := c.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return false, currentServerErr
	}

	// Check if the current profile supports OpenVPN
	profile, profileErr := server.CurrentProfile(currentServer)
	if profileErr != nil {
		return false, profileErr
	}

	if !profile.SupportsOpenVPN() {
		return false, errors.New("Profile does not support OpenVPN fallback")
	}

	c.Failover = failover.New(readRxBytes)

	return c.Failover.Start(gateway, wgMTU)
}

func (c *Client) CancelFailover() error {
	if c.Failover == nil {
		return errors.New("No failover process")
	}
	c.Failover.Cancel()
	return nil
}
