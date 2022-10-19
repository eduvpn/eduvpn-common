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
	client.FSM.GoTransition(STATE_REQUEST_CONFIG)

	validProfile, profileErr := server.HasValidProfile(chosenServer)
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
	return server.GetConfig(chosenServer, preferTCP)
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
		level := types.ERR_OTHER
		var error *oauth.OAuthTokensInvalidError
		var oauthCancelledError *oauth.OAuthCancelledCallbackError

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
		if errors.As(configErr, &oauthCancelledError) {
			level = types.ERR_INFO
		}
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Level: level, Message: errorMessage, Err: configErr}
	}
	return config, configType, nil
}

// getConfig gets an OpenVPN/WireGuard configuration by contacting the server, moving the FSM towards the DISCONNECTED state and then saving the local configuration file.
func (client *Client) getConfig(
	chosenServer server.Server,
	preferTCP bool,
) (string, string, error) {
	errorMessage := "failed to get a configuration for OpenVPN/Wireguard"
	if client.InFSMState(STATE_DEREGISTERED) {
		return "", "", &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}

	config, configType, configErr := client.retryConfigAuth(chosenServer, preferTCP)

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Level: types.GetErrorLevel(configErr), Message: errorMessage, Err: configErr}
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
	}

	// Signal the server display info
	client.FSM.GoTransitionWithData(STATE_DISCONNECTED, currentServer, false)

	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed saving configuration after getting a server: %s",
				types.GetErrorTraceback(saveErr),
			),
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
		return &types.WrappedErrorMessage{Message: errorMessage, Err: LetsConnectNotSupportedError{}}
	}

	server, serverErr := client.Discovery.GetServerByCountryCode(countryCode, "secure_internet")
	if serverErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed getting secure internet server by country code: %s with error: %s",
				countryCode,
				types.GetErrorTraceback(serverErr),
			),
		)
		client.goBackInternal()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	setLocationErr := client.Servers.SetSecureLocation(server)
	if setLocationErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed setting secure internet server with error: %s",
				types.GetErrorTraceback(setLocationErr),
			),
		)
		client.goBackInternal()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: setLocationErr}
	}
	return nil
}

// RemoveSecureInternet removes the current secure internet server.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveSecureInternet() error {
	if client.InFSMState(STATE_DEREGISTERED) {
		client.Logger.Error("Failed removing secure internet server due to deregistered")
		return &types.WrappedErrorMessage{
			Message: "failed to remove Secure Internet",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because we can only have one secure internet server and if there are no secure internet servers, this is a NO-OP
	client.Servers.RemoveSecureInternet()
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed saving configuration after removing a secure internet server: %s",
				types.GetErrorTraceback(saveErr),
			),
		)
	}
	return nil
}

// RemoveInstituteAccess removes the institute access server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveInstituteAccess(url string) error {
	if client.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: "failed to remove Institute Access",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because this is a NO-OP if the server doesn't exist
	client.Servers.RemoveInstituteAccess(url)
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed saving configuration after removing an institute access server: %s",
				types.GetErrorTraceback(saveErr),
			),
		)
	}
	return nil
}

// RemoveCustomServer removes the custom server with `url`.
// It returns an error if the server cannot be removed due to the state being DEREGISTERED.
// Note that if the server does not exist, it returns nil as an error.
func (client *Client) RemoveCustomServer(url string) error {
	if client.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: "failed to remove Custom Server",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because this is a NO-OP if the server doesn't exist
	client.Servers.RemoveCustomServer(url)
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"Failed saving configuration after removing a custom server: %s",
				types.GetErrorTraceback(saveErr),
			),
		)
	}
	return nil
}

// AddInstituteServer adds an Institute Access server by `url`.
func (client *Client) AddInstituteServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Institute Access server with url %s", url)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: LetsConnectNotSupportedError{}}
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(STATE_LOADING_SERVER)

	// FIXME: Do nothing with discovery here as the client already has it
	// So pass a server as the parameter
	instituteServer, discoErr := client.Discovery.GetServerByURL(url, "institute_access")
	if discoErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: discoErr}
	}

	// Add the secure internet server
	server, serverErr := client.Servers.AddInstituteAccessServer(instituteServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetInstituteAccess(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	// Indicate that we want to authorize this server
	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// Removing is best effort
		_ = client.RemoveInstituteAccess(url)
		return nil, &types.WrappedErrorMessage{Level: types.GetErrorLevel(loginErr), Message: errorMessage, Err: loginErr}
	}

	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
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
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: LetsConnectNotSupportedError{}}
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(STATE_LOADING_SERVER)

	// Get the secure internet URL from discovery
	secureOrg, secureServer, discoErr := client.Discovery.GetSecureHomeArgs(orgID)
	if discoErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: discoErr}
	}

	// Add the secure internet server
	server, serverErr := client.Servers.AddSecureInternet(secureOrg, secureServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	locationErr := client.askSecureLocation()
	if locationErr != nil {
		// Removing is best effort
		// This already goes back to the main screen
		_ = client.RemoveSecureInternet()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: locationErr}
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetSecureInternet(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil,  &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	// Server has been chosen for authentication
	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// Removing is best effort
		_ = client.RemoveSecureInternet()
		return nil, &types.WrappedErrorMessage{Level: types.GetErrorLevel(loginErr), Message: errorMessage, Err: loginErr}
	}
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
	return server, nil
}

// AddCustomServer adds a Custom Server by `url`
func (client *Client) AddCustomServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Custom server with url %s", url)

	url, urlErr := util.EnsureValidURL(url)
	if urlErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
	}

	// Indicate that we're loading the server
	client.FSM.GoTransition(STATE_LOADING_SERVER)

	customServer := &types.DiscoveryServer{
		BaseURL:     url,
		DisplayName: map[string]string{"en": url},
		Type:        "custom_server",
	}

	// A custom server is just an institute access server under the hood
	server, serverErr := client.Servers.AddCustomServer(customServer)
	if serverErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	// Set the server as the current so OAuth can be cancelled
	currentErr := client.Servers.SetCustomServer(server)
	if currentErr != nil {
		client.goBackInternal()
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	// Server has been chosen for authentication
	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	// Authorize it
	loginErr := client.ensureLogin(server)
	if loginErr != nil {
		// removing is best effort
		_ = client.RemoveCustomServer(url)
		return nil, &types.WrappedErrorMessage{Level: types.GetErrorLevel(loginErr), Message: errorMessage, Err: loginErr}
	}

	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)
	return server, nil
}

// GetConfigInstituteAccess gets a configuration for an Institute Access Server.
// It ensures that the Institute Access Server exists by creating or using an existing one with the url.
// `preferTCP` indicates that the client wants to use TCP (through OpenVPN) to establish the VPN tunnel.
func (client *Client) GetConfigInstituteAccess(url string, preferTCP bool) (string, string, error) {
	errorMessage := fmt.Sprintf("failed getting a configuration for Institute Access %s", url)

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: LetsConnectNotSupportedError{}}
	}

	client.FSM.GoTransition(STATE_LOADING_SERVER)

	// Get the server if it exists
	server, serverErr := client.Servers.GetInstituteAccess(url)
	if serverErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed getting an institute access server configuration with error: %s",
				types.GetErrorTraceback(serverErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	// Set the server as the current
	currentErr := client.Servers.SetInstituteAccess(server)
	if currentErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	// The server has now been chosen
	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.Logger.Inherit(configErr,
			fmt.Sprintf(
				"Failed getting an institute access server configuration with error: %s",
				types.GetErrorTraceback(configErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Level: types.GetErrorLevel(configErr), Message: errorMessage, Err: configErr}
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
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: LetsConnectNotSupportedError{}}
	}

	client.FSM.GoTransition(STATE_LOADING_SERVER)

	// Get the server if it exists
	server, serverErr := client.Servers.GetSecureInternetHomeServer()
	if serverErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed getting a custom server configuration with error: %s",
				types.GetErrorTraceback(serverErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	// Set the server as the current
	currentErr := client.Servers.SetSecureInternet(server)
	if currentErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.Logger.Inherit(
			configErr,
			fmt.Sprintf(
				"Failed getting a secure internet configuration with error: %s",
				types.GetErrorTraceback(configErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Level: types.GetErrorLevel(configErr), Message: errorMessage, Err: configErr}
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
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
	}

	client.FSM.GoTransition(STATE_LOADING_SERVER)

	// Get the server if it exists
	server, serverErr := client.Servers.GetCustomServer(url)
	if serverErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed getting a custom server configuration with error: %s",
				types.GetErrorTraceback(serverErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	// Set the server as the current
	currentErr := client.Servers.SetCustomServer(server)
	if currentErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: currentErr}
	}

	client.FSM.GoTransition(STATE_CHOSEN_SERVER)

	config, configType, configErr := client.getConfig(server, preferTCP)
	if configErr != nil {
		client.Logger.Inherit(
			configErr,
			fmt.Sprintf(
				"Failed getting a custom server with error: %s",
				types.GetErrorTraceback(configErr),
			),
		)
		client.goBackInternal()
		return "", "", &types.WrappedErrorMessage{Level: types.GetErrorLevel(configErr), Message: errorMessage, Err: configErr}
	}
	return config, configType, nil
}

// askSecureLocation asks the user to choose a Secure Internet location by moving the FSM to the STATE_ASK_LOCATION state.
func (client *Client) askSecureLocation() error {
	locations := client.Discovery.GetSecureLocationList()

	// Ask for the location in the callback
	client.FSM.GoTransitionWithData(STATE_ASK_LOCATION, locations, false)

	// The state has changed, meaning setting the secure location was not successful
	if client.FSM.Current != STATE_ASK_LOCATION {
		// TODO: maybe a custom type for this errors.new?
		return &types.WrappedErrorMessage{
			Message: "failed setting secure location",
			Err:     errors.New("failed loading secure location"),
		}
	}
	return nil
}

// ChangeSecureLocation changes the location for an existing Secure Internet Server.
// Changing a secure internet location is only possible when the user is in the main screen (STATE_NO_SERVER), otherwise it returns an error.
// It also returns an error if something has gone wrong when selecting the new location
func (client *Client) ChangeSecureLocation() error {
	errorMessage := "failed to change location from the main screen"

	if !client.InFSMState(STATE_NO_SERVER) {
		client.Logger.Error("Failed changing secure internet location, not in the right state")
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: FSMWrongStateError{
				Got:  client.FSM.Current,
				Want: STATE_NO_SERVER,
			}.CustomError(),
		}
	}

	askLocationErr := client.askSecureLocation()
	if askLocationErr != nil {
		client.Logger.Error(
			fmt.Sprintf(
				"Failed changing secure internet location, err: %s",
				types.GetErrorTraceback(askLocationErr),
			),
		)
		return &types.WrappedErrorMessage{Message: errorMessage, Err: askLocationErr}
	}

	// Go back to the main screen
	client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers, false)

	return nil
}


// RenewSession renews the session for the current VPN server.
// This logs the user back in.
func (client *Client) RenewSession() error {
	errorMessage := "failed to renew session"

	currentServer, currentServerErr := client.Servers.GetCurrentServer()
	if currentServerErr != nil {
		client.Logger.Warning(
			fmt.Sprintf(
				"Failed getting current server to renew, error: %s",
				types.GetErrorTraceback(currentServerErr),
			),
		)
		return &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
	}

	server.MarkTokensForRenew(currentServer)
	loginErr := client.ensureLogin(currentServer)
	if loginErr != nil {
		client.Logger.Warning(
			fmt.Sprintf(
				"Failed logging in server for renew, error: %s",
				types.GetErrorTraceback(loginErr),
			),
		)
		return &types.WrappedErrorMessage{Message: errorMessage, Err: loginErr}
	}

	return nil
}

// ShouldRenewButton returns true if the renew button should be shown
// If there is no server then this returns false and logs with INFO if so
// In other cases it simply checks the expiry time and calculates according to: https://github.com/eduvpn/documentation/blob/b93854dcdd22050d5f23e401619e0165cb8bc591/API.md#session-expiry.
func (client *Client) ShouldRenewButton() bool {
	if !client.InFSMState(STATE_CONNECTED) && !client.InFSMState(STATE_CONNECTING) &&
		!client.InFSMState(STATE_DISCONNECTED) &&
		!client.InFSMState(STATE_DISCONNECTING) {
		return false
	}

	currentServer, currentServerErr := client.Servers.GetCurrentServer()

	if currentServerErr != nil {
		client.Logger.Info(
			fmt.Sprintf(
				"No server found to renew with err: %s",
				types.GetErrorTraceback(currentServerErr),
			),
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
		url, urlErr := server.GetOAuthURL(chosenServer, client.Name)

		client.FSM.GoTransitionWithData(STATE_OAUTH_STARTED, url, true)

		if urlErr != nil {
			client.goBackInternal()
			return &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
		}

		exchangeErr := server.OAuthExchange(chosenServer)

		if exchangeErr != nil {
			client.goBackInternal()
			return &types.WrappedErrorMessage{Message: errorMessage, Err: exchangeErr}
		}
	}
	// OAuth was valid, ensure we are in the authorized state
	client.FSM.GoTransition(STATE_AUTHORIZED)
	return nil
}

// SetProfileID sets a `profileID` for the current server.
// An error is returned if this is not possible, for example when no server is configured.
func (client *Client) SetProfileID(profileID string) error {
	errorMessage := "failed to set the profile ID for the current server"
	server, serverErr := client.Servers.GetCurrentServer()
	if serverErr != nil {
		client.Logger.Warning(
			fmt.Sprintf(
				"Failed setting a profile ID because no server configured, Err: %s",
				types.GetErrorTraceback(serverErr),
			),
		)
		client.goBackInternal()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	base, baseErr := server.GetBase()
	if baseErr != nil {
		client.Logger.Error(
			fmt.Sprintf("Failed setting a profile ID, Err: %s", types.GetErrorTraceback(serverErr)),
		)
		client.goBackInternal()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	base.Profiles.Current = profileID
	return nil
}

