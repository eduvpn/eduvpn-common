package eduvpn

import (
	"errors"
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/config"
	"github.com/jwijenbergh/eduvpn-common/internal/discovery"
	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/oauth"
	"github.com/jwijenbergh/eduvpn-common/internal/server"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
)

type ServerInfo = server.ServerInfoScreen

type VPNState struct {
	// The chosen server
	Servers server.Servers `json:"servers"`

	// The list of servers and organizations from disco
	Discovery discovery.Discovery `json:"discovery"`

	// The fsm
	FSM fsm.FSM `json:"-"`

	// The logger
	Logger log.FileLogger `json:"-"`

	// The config
	Config config.Config `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

func (state *VPNState) GetSavedServers() *server.ServersConfiguredScreen {
	return state.Servers.GetServersConfigured()
}

func (state *VPNState) Register(
	name string,
	directory string,
	stateCallback func(FSMStateID, FSMStateID, interface{}),
	debug bool,
) error {
	errorMessage := "failed to register with the GO library"
	if !state.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// Initialize the logger
	logLevel := log.LOG_WARNING

	if debug {
		logLevel = log.LOG_INFO
	}

	loggerErr := state.Logger.Init(logLevel, name, directory)
	if loggerErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: loggerErr}
	}

	// Initialize the FSM
	state.FSM = newFSM(name, stateCallback, directory, debug)
	state.Debug = debug

	// Initialize the Config
	state.Config.Init(name, directory)

	// Try to load the previous configuration
	if state.Config.Load(&state) != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		state.Logger.Log(log.LOG_INFO, "Previous configuration not found")
	}

	discoServers, discoServersErr := state.GetDiscoServers()

	_, currentServerErr := state.Servers.GetCurrentServer()
	// TODO: Log the error always
	// Only actually return the error if we have no disco servers and no current server
	if discoServersErr != nil && discoServers == "" && currentServerErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: discoServersErr}
	}
	discoOrgs, discoOrgsErr := state.GetDiscoOrganizations()

	// TODO: Log the error always
	// Only actually return the error if we have no disco servers and no current server
	if discoOrgsErr != nil && discoOrgs == "" && currentServerErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: discoOrgsErr}
	}
	// Go to the No Server state with the saved servers
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), true)
	return nil
}

func (state *VPNState) Deregister() error {
	// Close the log file
	state.Logger.Close()

	// Save the config
	state.Config.Save(&state)

	// Empty out the state
	*state = VPNState{}
	return nil
}

func (state *VPNState) GoBack() error {
	errorMessage := "failed to go back"
	if state.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}

	// FIXME: Abitrary back transitions don't work because we need the approriate data
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), false)
	// state.FSM.GoBack()
	return nil
}

func (state *VPNState) doAuth(authURL string) error {
	state.FSM.GoTransitionWithData(STATE_OAUTH_STARTED, authURL, true)
	return nil
}

func (state *VPNState) ensureLogin(chosenServer server.Server) error {
	errorMessage := "failed ensuring login"
	// Relogin with oauth
	// This moves the state to authorized
	if server.NeedsRelogin(chosenServer) {
		url, urlErr := server.GetOAuthURL(chosenServer, state.FSM.Name)

		state.FSM.GoTransitionWithData(STATE_OAUTH_STARTED, url, true)

		if urlErr != nil {
			state.GoBack()
			return &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
		}

		exchangeErr := server.OAuthExchange(chosenServer)

		if exchangeErr != nil {
			state.GoBack()
			return &types.WrappedErrorMessage{Message: errorMessage, Err: exchangeErr}
		}
	}
	// OAuth was valid, ensure we are in the authorized state
	state.FSM.GoTransition(STATE_AUTHORIZED)
	return nil
}

func (state *VPNState) getConfigAuth(chosenServer server.Server, forceTCP bool) (string, string, error) {
	loginErr := state.ensureLogin(chosenServer)
	if loginErr != nil {
		return "", "", loginErr
	}
	state.FSM.GoTransition(STATE_REQUEST_CONFIG)

	validProfile, profileErr := server.HasValidProfile(chosenServer)
	if profileErr != nil {
		return "", "", profileErr
	}

	// No valid profile, ask for one
	if !validProfile {
		askProfileErr := state.askProfile(chosenServer)
		if askProfileErr != nil {
			return "", "", askProfileErr
		}
	}

	// We return the error otherwise we wrap it too much
	return server.GetConfig(chosenServer, forceTCP)
}

func (state *VPNState) retryConfigAuth(chosenServer server.Server, forceTCP bool) (string, string, error) {
	errorMessage := "failed authorized config retry"
	config, configType, configErr := state.getConfigAuth(chosenServer, forceTCP)
	if configErr != nil {
		var error *oauth.OAuthTokensInvalidError

		// Only retry if the error is that the tokens are invalid
		if errors.As(configErr, &error) {
			retryConfig, retryConfigType, retryConfigErr := state.getConfigAuth(chosenServer, forceTCP)
			if retryConfigErr != nil {
				state.GoBack()
				return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: retryConfigErr}
			}
			return retryConfig, retryConfigType, nil
		}
		state.GoBack()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}
	return config, configType, nil
}


func (state *VPNState) getConfig(
	chosenServer server.Server,
	forceTCP bool,
) (string, string, error) {
	errorMessage := "failed to get a configuration for OpenVPN/Wireguard"
	if state.InFSMState(STATE_DEREGISTERED) {
		return "", "", &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}

	config, configType, configErr := state.retryConfigAuth(chosenServer, forceTCP)

	if configErr != nil {
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: configErr}
	}

	// Signal the server display info
	state.FSM.GoTransitionWithData(STATE_DISCONNECTED, state.getServerInfoData(), false)

	// Save the config
	state.Config.Save(&state)

	return config, configType, nil
}

func (state *VPNState) SetSecureLocation(countryCode string) error {
	errorMessage := "failed asking secure location"

	server, serverErr := state.Discovery.GetServerByCountryCode(countryCode, "secure_internet")
	if serverErr != nil {
		state.GoBack()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	setLocationErr := state.Servers.SetSecureLocation(server)
	if setLocationErr != nil {
		state.GoBack()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: setLocationErr}
	}
	return nil
}

func (state *VPNState) askProfile(chosenServer server.Server) error {
	base, baseErr := chosenServer.GetBase()
	if baseErr != nil {
		return &types.WrappedErrorMessage{Message: "failed asking for profiles", Err: baseErr}
	}
	state.FSM.GoTransitionWithData(STATE_ASK_PROFILE, &base.Profiles, false)
	return nil
}

func (state *VPNState) askSecureLocation() error {
	locations := state.Discovery.GetSecureLocationList()

	// Ask for the location in the callback
	state.FSM.GoTransitionWithData(STATE_ASK_LOCATION, locations, false)

	// The state has changed, meaning setting the secure location was not successful
	if state.FSM.Current != STATE_ASK_LOCATION {
		// TODO: maybe a custom type for this errors.new?
		return &types.WrappedErrorMessage{Message: "failed setting secure location", Err: errors.New("failed loading secure location")}
	}
	return nil
}

func (state *VPNState) addSecureInternetHomeServer(orgID string) (server.Server, error) {
	errorMessage := fmt.Sprintf(
		"failed adding Secure Internet home server with organization ID %s",
		orgID,
	)
	// Get the secure internet URL from discovery
	secureOrg, secureServer, discoErr := state.Discovery.GetSecureHomeArgs(orgID)
	if discoErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: discoErr}
	}

	// Add the secure internet server
	server, serverErr := state.Servers.AddSecureInternet(secureOrg, secureServer)

	if serverErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	var locationErr error

	if !state.Servers.HasSecureLocation() {
		locationErr = state.askSecureLocation()
	} else {
		// reinitialize
		locationErr = state.SetSecureLocation(state.Servers.GetSecureLocation())
	}

	if locationErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: locationErr}
	}

	return server, nil
}

func (state *VPNState) RemoveSecureInternet() error {
	if state.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: "failed to remove Secure Internet",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because we can only have one secure internet server and if there are no secure internet servers, this is a NO-OP
	state.Servers.RemoveSecureInternet()
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), false)
	// Save the config
	state.Config.Save(&state)
	return nil
}

func (state *VPNState) RemoveInstituteAccess(url string) error {
	if state.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: "failed to remove Institute Access",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because this is a NO-OP if the server doesn't exist
	state.Servers.RemoveInstituteAccess(url)
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), false)
	// Save the config
	state.Config.Save(&state)
	return nil
}

func (state *VPNState) RemoveCustomServer(url string) error {
	if state.InFSMState(STATE_DEREGISTERED) {
		return &types.WrappedErrorMessage{
			Message: "failed to remove Custom Server",
			Err:     FSMDeregisteredError{}.CustomError(),
		}
	}
	// No error because this is a NO-OP if the server doesn't exist
	state.Servers.RemoveCustomServer(url)
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), false)
	// Save the config
	state.Config.Save(&state)
	return nil
}

func (state *VPNState) GetConfigSecureInternet(
	orgID string,
	forceTCP bool,
) (string, string, error) {
	errorMessage := fmt.Sprintf(
		"failed getting a configuration for Secure Internet organization %s",
		orgID,
	)
	state.FSM.GoTransition(STATE_LOADING_SERVER)
	server, serverErr := state.addSecureInternetHomeServer(orgID)

	if serverErr != nil {
		state.GoBack()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	state.FSM.GoTransition(STATE_CHOSEN_SERVER)

	return state.getConfig(server, forceTCP)
}

func (state *VPNState) addInstituteServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Institute Access server with url %s", url)
	instituteServer, discoErr := state.Discovery.GetServerByURL(url, "institute_access")
	if discoErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: discoErr}
	}
	// Add the secure internet server
	server, serverErr := state.Servers.AddInstituteAccessServer(instituteServer)

	if serverErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	state.FSM.GoTransition(STATE_CHOSEN_SERVER)

	return server, nil
}

func (state *VPNState) addCustomServer(url string) (server.Server, error) {
	errorMessage := fmt.Sprintf("failed adding Custom server with url %s", url)

	url, urlErr := util.EnsureValidURL(url)

	if urlErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: urlErr}
	}

	customServer := &types.DiscoveryServer{
		BaseURL:     url,
		DisplayName: map[string]string{"en": url},
		Type:        "custom_server",
	}

	// A custom server is just an institute access server under the hood
	server, serverErr := state.Servers.AddCustomServer(customServer)

	if serverErr != nil {
		return nil, &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	state.FSM.GoTransition(STATE_CHOSEN_SERVER)

	return server, nil
}

func (state *VPNState) GetConfigInstituteAccess(url string, forceTCP bool) (string, string, error) {
	errorMessage := fmt.Sprintf("failed getting a configuration for Institute Access %s", url)
	state.FSM.GoTransition(STATE_LOADING_SERVER)
	server, serverErr := state.addInstituteServer(url)

	if serverErr != nil {
		state.GoBack()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	return state.getConfig(server, forceTCP)
}

func (state *VPNState) GetConfigCustomServer(url string, forceTCP bool) (string, string, error) {
	errorMessage := fmt.Sprintf("failed getting a configuration for custom server %s", url)
	state.FSM.GoTransition(STATE_LOADING_SERVER)
	server, serverErr := state.addCustomServer(url)

	if serverErr != nil {
		state.GoBack()
		return "", "", &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	return state.getConfig(server, forceTCP)
}

func (state *VPNState) CancelOAuth() error {
	errorMessage := "failed to cancel OAuth"
	if !state.InFSMState(STATE_OAUTH_STARTED) {
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: FSMWrongStateError{
				Got:  state.FSM.Current,
				Want: STATE_OAUTH_STARTED,
			}.CustomError(),
		}
	}

	currentServer, serverErr := state.Servers.GetCurrentServer()

	if serverErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}
	server.CancelOAuth(currentServer)
	return nil
}

func (state *VPNState) ChangeSecureLocation() error {
	errorMessage := "failed to change location from the main screen"

	if !state.InFSMState(STATE_NO_SERVER) {
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err:     FSMWrongStateError{Got: state.FSM.Current, Want: STATE_NO_SERVER}.CustomError(),
		}
	}

	askLocationErr := state.askSecureLocation()

	if askLocationErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: askLocationErr}
	}

	// Go back to the main screen
	state.FSM.GoTransitionWithData(STATE_NO_SERVER, state.GetSavedServers(), false)

	return nil
}

func (state *VPNState) GetDiscoOrganizations() (string, error) {
	return state.Discovery.GetOrganizationsList()
}

func (state *VPNState) GetDiscoServers() (string, error) {
	return state.Discovery.GetServersList()
}

func (state *VPNState) SetProfileID(profileID string) error {
	errorMessage := "failed to set the profile ID for the current server"
	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		state.GoBack()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: serverErr}
	}

	base, baseErr := server.GetBase()

	if baseErr != nil {
		state.GoBack()
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}
	base.Profiles.Current = profileID
	return nil
}

func (state *VPNState) SetSearchServer() error {
	if !state.FSM.HasTransition(STATE_SEARCH_SERVER) {
		return &types.WrappedErrorMessage{
			Message: "failed to set search server",
			Err: FSMWrongStateTransitionError{
				Got:  state.FSM.Current,
				Want: STATE_CONNECTED,
			}.CustomError(),
		}
	}

	state.FSM.GoTransition(STATE_SEARCH_SERVER)
	return nil
}

func (state *VPNState) getServerInfoData() *server.ServerInfoScreen {
	info, _ := state.Servers.GetCurrentServerInfo()
	// TODO: Log error
	return info
}

func (state *VPNState) SetConnected() error {
	if state.InFSMState(STATE_CONNECTED) {
		// already connected, show no error
		return nil
	}
	if !state.FSM.HasTransition(STATE_CONNECTED) {
		return &types.WrappedErrorMessage{
			Message: "failed to set connected",
			Err: FSMWrongStateTransitionError{
				Got:  state.FSM.Current,
				Want: STATE_CONNECTED,
			}.CustomError(),
		}
	}

	state.FSM.GoTransitionWithData(STATE_CONNECTED, state.getServerInfoData(), false)
	return nil
}

func (state *VPNState) SetConnecting() error {
	if state.InFSMState(STATE_CONNECTING) {
		// already loading connection, show no error
		return nil
	}
	if !state.FSM.HasTransition(STATE_CONNECTING) {
		return &types.WrappedErrorMessage{
			Message: "failed to set connecting",
			Err: FSMWrongStateTransitionError{
				Got:  state.FSM.Current,
				Want: STATE_CONNECTING,
			}.CustomError(),
		}
	}

	state.FSM.GoTransition(STATE_CONNECTING)
	return nil
}

func (state *VPNState) SetDisconnecting() error {
	if state.InFSMState(STATE_DISCONNECTING) {
		// already disconnecting, show no error
		return nil
	}
	if !state.FSM.HasTransition(STATE_DISCONNECTING) {
		return &types.WrappedErrorMessage{
			Message: "failed to set disconnecting",
			Err: FSMWrongStateTransitionError{
				Got:  state.FSM.Current,
				Want: STATE_DISCONNECTING,
			}.CustomError(),
		}
	}

	state.FSM.GoTransitionWithData(STATE_DISCONNECTING, state.getServerInfoData(), false)
	return nil
}

func (state *VPNState) SetDisconnected(cleanup bool) error {
	errorMessage := "failed to set disconnected"
	if state.InFSMState(STATE_DISCONNECTED) {
		// already disconnected, show no error
		return nil
	}
	if !state.FSM.HasTransition(STATE_DISCONNECTED) {
		return &types.WrappedErrorMessage{
			Message: errorMessage,
			Err: FSMWrongStateTransitionError{
				Got:  state.FSM.Current,
				Want: STATE_DISCONNECTED,
			}.CustomError(),
		}
	}

	if cleanup {
		// Do the /disconnect API call and go to disconnected after...
		currentServer, currentServerErr := state.Servers.GetCurrentServer()
		if currentServerErr != nil {
			return &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
		}

		server.Disconnect(currentServer)
	}

	state.FSM.GoTransitionWithData(STATE_DISCONNECTED, state.getServerInfoData(), false)

	return nil
}

func (state *VPNState) RenewSession() error {
	errorMessage := "failed to renew session"

	currentServer, currentServerErr := state.Servers.GetCurrentServer()

	if currentServerErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: currentServerErr}
	}

	// FIXME: Delete tokens?

	loginErr := state.ensureLogin(currentServer)
	if loginErr != nil {
		// Go back
		return &types.WrappedErrorMessage{Message: errorMessage, Err: loginErr}
	}

	return nil
}

func (state *VPNState) ShouldRenewButton() bool {
	if !state.InFSMState(STATE_CONNECTED) && !state.InFSMState(STATE_CONNECTING) && !state.InFSMState(STATE_DISCONNECTED) && !state.InFSMState(STATE_DISCONNECTING) {
		return false
	}

	currentServer, currentServerErr := state.Servers.GetCurrentServer()

	if currentServerErr != nil {
		state.Logger.Log(
			log.LOG_INFO,
			fmt.Sprintf(
				"No server found to renew with err: %s",
				GetErrorTraceback(currentServerErr),
			),
		)
		return false
	}

	return server.ShouldRenewButton(currentServer)
}

func (state *VPNState) InFSMState(checkState FSMStateID) bool {
	return state.FSM.InState(checkState)
}

func GetErrorCause(err error) error {
	return types.GetErrorCause(err)
}

func GetErrorLevel(err error) types.ErrorLevel {
	return types.GetErrorLevel(err)
}

func GetErrorTraceback(err error) string {
	return types.GetErrorTraceback(err)
}

func GetErrorJSONString(err error) string {
	return types.GetErrorJSONString(err)
}
