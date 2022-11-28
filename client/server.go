package client

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/types"
	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/internal/util"
)

// getConfigAuth gets a config with authorization and authentication.
// It also asks for a profile if no valid profile is found.
func (client *Client) getConfigAuth(
	chosenServer server.Server,
	preferTCP bool,
) (string, string, error) {
	loginErr := client.ensureLogin(chosenServer)
	if loginErr != nil {
		return "", "", loginErr
	}
	client.FSM.GoTransition(StateRequestConfig)

	validProfile, profileErr := server.HasValidProfile(chosenServer, client.SupportsWireguard)
	if profileErr != nil {
		return "", "", profileErr
	}

	// No valid profile, ask for one
	if !validProfile {
		askProfileErr := client.askProfile(chosenServer)
		if askProfileErr != nil {
			return "", "", askProfileErr
		}
	}

	// We return the error otherwise we wrap it too much
	return server.Config(chosenServer, client.SupportsWireguard, preferTCP)
}

// retryConfigAuth retries the getConfigAuth function if the tokens are invalid.
// If OAuth is cancelled, it makes sure that we only forward the error as additional info.
func (client *Client) retryConfigAuth(
	chosenServer server.Server,
	preferTCP bool,
) (string, string, error) {
	errorMessage := "failed authorized config retry"
	config, configType, configErr := client.getConfigAuth(chosenServer, preferTCP)
	if configErr != nil {
		var error *oauth.OAuthTokensInvalidError

		// Only retry if the error is that the tokens are invalid
		if errors.As(configErr, &error) {
			config, configType, configErr = client.getConfigAuth(
				chosenServer,
				preferTCP,
			)
			if configErr == nil {
				return config, configType, nil
			}
		}
		client.goBackInternal()
		return "", "", types.NewWrappedError(errorMessage, configErr)
	}
	return config, configType, nil
}

// getConfig gets an OpenVPN/WireGuard configuration by contacting the server, moving the FSM towards the DISCONNECTED state and then saving the local configuration file.
func (client *Client) getConfig(
	chosenServer server.Server,
	preferTCP bool,
) (string, string, error) {
	errorMessage := "failed to get a configuration for OpenVPN/Wireguard"
	if client.InFSMState(StateDeregistered) {
		return "", "", types.NewWrappedError(
			errorMessage,
			FSMDeregisteredError{}.CustomError(),
		)
	}

	// Refresh the server endpoints
	// This is best effort
	endpointErr := server.RefreshEndpoints(chosenServer)
	if endpointErr != nil {
		client.Logger.Warning("failed to refresh server endpoints: %v", endpointErr)
	}

	config, configType, configErr := client.retryConfigAuth(chosenServer, preferTCP)
	if configErr != nil {
		return "", "", types.NewWrappedError(errorMessage, configErr)
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return "", "", types.NewWrappedError(errorMessage, currentServerErr)
	}

	// Signal the server display info
	client.FSM.GoTransitionWithData(StateDisconnected, currentServer)

	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			"Failed saving configuration after getting a server: %s",
			types.ErrorTraceback(saveErr),
		)
	}

	return config, configType, nil
}

// SetSecureLocation sets the location for the current secure location server. countryCode is the secure location to be chosen.
// This function returns an error e.g. if the server cannot be found or the location is wrong.
func (client *Client) SetSecureLocation(countryCode string) error {
	errorMessage := "failed asking secure location"

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	server, serverErr := client.Discovery.ServerByCountryCode(countryCode, "secure_internet")
	if serverErr != nil {
		client.goBackInternal()
		return client.handleError(errorMessage, serverErr)
	}

	setLocationErr := client.Servers.SetSecureLocation(server)
	if setLocationErr != nil {
		client.goBackInternal()
		return client.handleError(errorMessage, setLocationErr)
	}
	return nil
}

// RemoveSecureInternet removes the current secure internet server.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveSecureInternet() error {
	if client.InFSMState(StateDeregistered) {
		return client.handleError(
			"failed to remove Secure Internet",
			FSMDeregisteredError{}.CustomError(),
		)
	}
	// No error because we can only have one secure internet server and if there are no secure internet servers, this is a NO-OP
	client.Servers.RemoveSecureInternet()
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			"Failed saving configuration after removing a secure internet server: %s",
			types.ErrorTraceback(saveErr),
		)
	}
	return nil
}

// RemoveInstituteAccess removes the institute access server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveInstituteAccess(url string) error {
	if client.InFSMState(StateDeregistered) {
		return client.handleError(
			"failed to remove Institute Access",
			FSMDeregisteredError{}.CustomError(),
		)
	}
	// No error because this is a NO-OP if the server doesn't exist
	client.Servers.RemoveInstituteAccess(url)
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			"Failed saving configuration after removing an institute access server: %s",
			types.ErrorTraceback(saveErr),
		)
	}
	return nil
}

// RemoveCustomServer removes the custom server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveCustomServer(url string) error {
	if client.InFSMState(StateDeregistered) {
		return client.handleError(
			"failed to remove Custom Server",
			FSMDeregisteredError{}.CustomError(),
		)
	}
	// No error because this is a NO-OP if the server doesn't exist
	client.Servers.RemoveCustomServer(url)
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			"Failed saving configuration after removing a custom server: %s",
			types.ErrorTraceback(saveErr),
		)
	}
	return nil
}

// AddInstituteServer adds an Institute Access server by `url`.
func (client *Client) AddInstituteServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Institute Access server with url %s", url)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return nil, client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(StateLoadingServer)

	// FIXME: Do nothing with discovery here as the client already has it
	// So pass a server as the parameter
	instituteServer, discoErr := client.Discovery.ServerByURL(url, "institute_access")
	if discoErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, discoErr)
	}

	// Add the secure internet server
	server, serverErr := client.Servers.AddInstituteAccessServer(instituteServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, serverErr)
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetInstituteAccess(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, currentErr)
	}

	// Indicate that we want to authorize this server
	client.FSM.GoTransition(StateChosenServer)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// Removing is best effort
		_ = client.RemoveInstituteAccess(url)
		return nil, client.handleError(errorMessage, loginErr)
	}

	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	return server, nil
}

// AddSecureInternetHomeServer adds a Secure Internet Home Server with `orgID` that was obtained from the Discovery file.
// Because there is only one Secure Internet Home Server, it replaces the existing one.
func (client *Client) AddSecureInternetHomeServer(orgID string) (server.Server, error) {
	errorMessage := fmt.Sprintf(
		"failed adding Secure Internet home server with organization ID %s",
		orgID,
	)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return nil, client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(StateLoadingServer)

	// Get the secure internet URL from discovery
	secureOrg, secureServer, discoErr := client.Discovery.SecureHomeArgs(orgID)
	if discoErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, discoErr)
	}

	// Add the secure internet server
	server, serverErr := client.Servers.AddSecureInternet(secureOrg, secureServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, serverErr)
	}

	locationErr := client.askSecureLocation()
	if locationErr != nil {
		// Removing is best effort
		// This already goes back to the main screen
		_ = client.RemoveSecureInternet()
		return nil, client.handleError(errorMessage, locationErr)
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetSecureInternet(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil,  client.handleError(errorMessage, currentErr)
	}

	// Server has been chosen for authentication
	client.FSM.GoTransition(StateChosenServer)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// Removing is best effort
		_ = client.RemoveSecureInternet()
		return nil, client.handleError(errorMessage, loginErr)
	}
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	return server, nil
}

// AddCustomServer adds a Custom Server by `url`
func (client *Client) AddCustomServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Custom server with url %s", url)

	url, urlErr := util.EnsureValidURL(url)
	if urlErr != nil {
		return nil, client.handleError(errorMessage, urlErr)
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(StateLoadingServer)

	customServer := &types.DiscoveryServer{
		BaseURL:     url,
		DisplayName: map[string]string{"en": url},
		Type:        "custom_server",
	}

	// A custom server is just an institute access server under the hood
	server, serverErr := client.Servers.AddCustomServer(customServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, serverErr)
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetCustomServer(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil, client.handleError(errorMessage, currentErr)
	}

	// Server has been chosen for authentication
	client.FSM.GoTransition(StateChosenServer)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// removing is best effort
		_ = client.RemoveCustomServer(url)
		return nil, client.handleError(errorMessage, loginErr)
	}

	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)
	return server, nil
}

// GetConfigInstituteAccess gets a configuration for an Institute Access Server.
// It ensures that the Institute Access Server exists by creating or using an existing one with the url.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (client *Client) GetConfigInstituteAccess(url string, preferTCP bool) (string, string, error) {
	errorMessage := fmt.Sprintf("failed getting a configuration for Institute Access %s", url)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return "", "", client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	client.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	server, serverErr := client.Servers.GetInstituteAccess(url)
	if serverErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, serverErr)
	}

	// Set the server as the current
	currentErr := client.Servers.SetInstituteAccess(server)
	if currentErr != nil {
		return "", "", client.handleError(errorMessage, currentErr)
	}

	// The server has now been chosen
	client.FSM.GoTransition(StateChosenServer)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, configErr)
	}
	return config, configType, nil
}

// GetConfigSecureInternet gets a configuration for a Secure Internet Server.
// It ensures that the Secure Internet Server exists by creating or using an existing one with the orgID.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (client *Client) GetConfigSecureInternet(
	orgID string,
	preferTCP bool,
) (string, string, error) {
	errorMessage := fmt.Sprintf(
		"failed getting a configuration for Secure Internet organization %s",
		orgID,
	)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return "", "", client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	client.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	server, serverErr := client.Servers.GetSecureInternetHomeServer()
	if serverErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, serverErr)
	}

	// Set the server as the current
	currentErr := client.Servers.SetSecureInternet(server)
	if currentErr != nil {
		return "", "", client.handleError(errorMessage, currentErr)
	}

	client.FSM.GoTransition(StateChosenServer)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, configErr)
	}
	return config, configType, nil
}

// GetConfigCustomServer gets a configuration for a Custom Server.
// It ensures that the Custom Server exists by creating or using an existing one with the url.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (client *Client) GetConfigCustomServer(url string, preferTCP bool) (string, string, error) {
	errorMessage := fmt.Sprintf("failed getting a configuration for custom server %s", url)

	url, urlErr := util.EnsureValidURL(url)
	if urlErr != nil {
		return "", "", client.handleError(errorMessage, urlErr)
	}

	client.FSM.GoTransition(StateLoadingServer)

	// Get the server if it exists
	server, serverErr := client.Servers.GetCustomServer(url)
	if serverErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, serverErr)
	}

	// Set the server as the current
	currentErr := client.Servers.SetCustomServer(server)
	if currentErr != nil {
		return "", "", client.handleError(errorMessage, currentErr)
	}

	client.FSM.GoTransition(StateChosenServer)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.goBackInternal()
		return "", "", client.handleError(errorMessage, configErr)
	}
	return config, configType, nil
}

// askSecureLocation asks the user to choose a Secure Internet location by moving the FSM to the STATE_ASK_LOCATION state.
func (client *Client) askSecureLocation() error {
	errorMessage := "failed settings secure location"
	locations := client.Discovery.SecureLocationList()

	// Ask for the location in the callback
	goTransitionErr := client.FSM.GoTransitionRequired(StateAskLocation, locations)
	if goTransitionErr != nil {
		return types.NewWrappedError(errorMessage, goTransitionErr)
	}

	// The state has changed, meaning setting the secure location was not successful
	if client.FSM.Current != StateAskLocation {
		// TODO: maybe a custom type for this errors.new?
		return types.NewWrappedError(
			errorMessage,
			errors.New("failed loading secure location"),
		)
	}
	return nil
}

// ChangeSecureLocation changes the location for an existing Secure Internet Server.
// Changing a secure internet location is only possible when the user is in the main screen (STATE_NO_SERVER), otherwise it returns an error.
// It also returns an error if something has gone wrong when selecting the new location
func (client *Client) ChangeSecureLocation() error {
	errorMessage := "failed to change location from the main screen"

	if !client.InFSMState(StateNoServer) {
		return client.handleError(
			errorMessage,
			FSMWrongStateError{
				Got:  client.FSM.Current,
				Want: StateNoServer,
			}.CustomError(),
		)
	}

	askLocationErr := client.askSecureLocation()
	if askLocationErr != nil {
		return client.handleError(errorMessage, askLocationErr)
	}

	// Go back to the main screen
	client.FSM.GoTransitionWithData(StateNoServer, client.Servers)

	return nil
}


// RenewSession renews the session for the current VPN server.
// This logs the user back in.
func (client *Client) RenewSession() error {
	errorMessage := "failed to renew session"

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return client.handleError(errorMessage, currentServerErr)
	}

	// The server has not been chosen yet, this means that we want to manually renew
	if client.FSM.InState(StateNoServer) {
	    client.FSM.GoTransition(StateChosenServer)
	}

	server.MarkTokensForRenew(currentServer)
	loginErr := client.ensureLogin(currentServer)
	if loginErr != nil {
		return client.handleError(errorMessage, loginErr)
	}

	return nil
}

// ShouldRenewButton returns true if the renew button should be shown
// If there is no server then this returns false and logs with INFO if so
// In other cases it simply checks the expiry time and calculates according to: https://github.com/eduvpn/documentation/blob/b93854dcdd22050d5f23e401619e0165cb8bc591/API.md#session-expiry.
func (client *Client) ShouldRenewButton() bool {
	if !client.InFSMState(StateConnected) && !client.InFSMState(StateConnecting) &&
		!client.InFSMState(StateDisconnected) &&
		!client.InFSMState(StateDisconnecting) {
		return false
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()

	if currentServerErr != nil {
		client.Logger.Info(
			"No server found to renew with err: %s",
			types.ErrorTraceback(currentServerErr),
		)
		return false
	}

	return server.ShouldRenewButton(currentServer)
}

// ensureLogin logs the user back in if needed.
// It runs the FSM transitions to ask for user input.
func (client *Client) ensureLogin(chosenServer server.Server) error {
	errorMessage := "failed ensuring login"
	// Relogin with oauth
	// This moves the state to authorized
	if server.NeedsRelogin(chosenServer) {
		url, urlErr := server.OAuthURL(chosenServer, client.Name)

		goTransitionErr := client.FSM.GoTransitionRequired(StateOAuthStarted, url)
		if goTransitionErr != nil {
			return types.NewWrappedError(errorMessage, goTransitionErr)
		}

		if urlErr != nil {
			client.goBackInternal()
			return types.NewWrappedError(errorMessage, urlErr)
		}

		exchangeErr := server.OAuthExchange(chosenServer)

		if exchangeErr != nil {
			client.goBackInternal()
			return types.NewWrappedError(errorMessage, exchangeErr)
		}
	}
	// OAuth was valid, ensure we are in the authorized state
	client.FSM.GoTransition(StateAuthorized)
	return nil
}

// SetProfileID sets a `profileID` for the current server.
// An error is returned if this is not possible, for example when no server is configured.
func (client *Client) SetProfileID(profileID string) error {
	errorMessage := "failed to set the profile ID for the current server"
	server, serverErr := client.Servers.GetCurrentServer()
	if serverErr != nil {
		client.goBackInternal()
		return client.handleError(errorMessage, serverErr)
	}

	base, baseErr := server.Base()
	if baseErr != nil {
		client.goBackInternal()
		return client.handleError(errorMessage, baseErr)
	}
	base.Profiles.Current = profileID
	return nil
}

