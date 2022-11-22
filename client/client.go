package client

import (
	"strings"

	"github.com/eduvpn/eduvpn-common/internal/config"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/fsm"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/internal/server"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types"
)

type (
	// ServerBase is an alias to the internal ServerBase
	// This contains the details for each server
	ServerBase = server.ServerBase
)

// This wraps the error, logs it and then returns the wrapped error
func (client *Client) handleError(message string, err error) error {
	if err != nil {
		// Logs the error with the same level/verbosity as the error
		client.Logger.Inherit(message, err)
		return types.NewWrappedError(message, err)
	}
	return nil
}

func (client Client) isLetsConnect() bool {
	// see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/ClientDb.php
	return strings.HasPrefix(client.Name, "org.letsconnect-vpn.app")
}

// Client is the main struct for the VPN client
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

	// The logger
	Logger log.FileLogger `json:"-"`

	// The config
	Config config.Config `json:"-"`

	// Whether or not this client supports WireGuard
	SupportsWireguard bool `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

// Register initializes the clientwith the following parameters:
//  - name: the name of the client
//  - directory: the directory where the config files are stored. Absolute or relative
//  - stateCallback: the callback function for the FSM that takes two states (old and new) and the data as an interface
//  - debug: whether or not we want to enable debugging
// It returns an error if initialization failed, for example when discovery cannot be obtained and when there are no servers.
func (client *Client) Register(
	name string,
	directory string,
	language string,
	stateCallback func(FSMStateID, FSMStateID, interface{}) bool,
	debug bool,
) error {
	errorMessage := "failed to register with the GO library"
	if !client.InFSMState(STATE_DEREGISTERED) {
		return client.handleError(
			errorMessage,
			FSMDeregisteredError{}.CustomError(),
		)
	}
	client.Name = name

	// TODO: Verify language setting?
	client.Language = language

	// Initialize the logger
	logLevel := log.LOG_WARNING
	if debug {
		logLevel = log.LOG_DEBUG
	}

	loggerErr := client.Logger.Init(logLevel, directory)
	if loggerErr != nil {
		return client.handleError(errorMessage, loggerErr)
	}

	// Initialize the FSM
	client.FSM = newFSM(stateCallback, directory, debug)

	// By default we support wireguard
	client.SupportsWireguard = true

	// Debug only if given
	client.Debug = debug

	// Initialize the Config
	client.Config.Init(directory, "state")

	// Try to load the previous configuration
	if client.Config.Load(&client) != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		client.Logger.Info("Previous configuration not found")
	}

	// Go to the No Server state with the saved servers after we're done
	defer client.FSM.GoTransitionWithData(STATE_NO_SERVER, client.Servers)

	// Let's Connect! doesn't care about discovery
	if client.isLetsConnect() {
		return nil
	}

	// Check if we are able to fetch discovery, and log if something went wrong
	_, discoServersErr := client.GetDiscoServers()
	if discoServersErr != nil {
		client.Logger.Warning("Failed to get discovery servers: %v", discoServersErr)
	}
	_, discoOrgsErr := client.GetDiscoOrganizations()
	if discoOrgsErr != nil {
		client.Logger.Warning("Failed to get discovery organizations: %v", discoOrgsErr)
	}

	return nil
}

// Deregister 'deregisters' the client, meaning saving the log file and the config and emptying out the client struct.
func (client *Client) Deregister() {
	// Close the log file
	client.Logger.Close()

	// Save the config
	saveErr := client.Config.Save(&client)
	if saveErr != nil {
		client.Logger.Info("failed saving configuration, error: %s", types.GetErrorTraceback(saveErr))
	}

	// Empty out the state
	*client = Client{}
}

// askProfile asks the user for a profile by moving the FSM to the ASK_PROFILE state.
func (client *Client) askProfile(chosenServer server.Server) error {
	errorMessage := "failed asking for profiles"
	profiles, profilesErr := server.GetValidProfiles(chosenServer, client.SupportsWireguard)
	if profilesErr != nil {
		return types.NewWrappedError(errorMessage, profilesErr)
	}
	goTransitionErr := client.FSM.GoTransitionRequired(STATE_ASK_PROFILE, profiles)
	if goTransitionErr != nil {
		return types.NewWrappedError(errorMessage, goTransitionErr)
	}
	return nil
}

// GetDiscoOrganizations gets the organizations list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#organization-list.
func (client *Client) GetDiscoOrganizations() (*types.DiscoveryOrganizations, error) {
	errorMessage := "failed getting discovery organizations list"
	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return nil, client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	orgs, orgsErr := client.Discovery.GetOrganizationsList()
	if orgsErr != nil {
		return nil, client.handleError(
			errorMessage,
			orgsErr,
		)
	}
	return orgs, nil
}

// GetDiscoServers gets the servers list from the discovery server
// If the list cannot be retrieved an error is returned.
// If this is the case then a previous version of the list is returned if there is any.
// This takes into account the frequency of updates, see: https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md#server-list.
func (client *Client) GetDiscoServers() (*types.DiscoveryServers, error) {
	errorMessage := "failed getting discovery servers list"

	// Not supported with Let's Connect!
	if client.isLetsConnect() {
		return nil, client.handleError(errorMessage, LetsConnectNotSupportedError{})
	}

	servers, serversErr := client.Discovery.GetServersList()
	if serversErr != nil {
		return nil, client.handleError(
			errorMessage,
			serversErr,
		)
	}
	return servers, nil
}

// GetTranslated gets the translation for `languages` using the current state language.
func (client *Client) GetTranslated(languages map[string]string) string {
	return util.GetLanguageMatched(languages, client.Language)
}

type LetsConnectNotSupportedError struct{}

func (e LetsConnectNotSupportedError) Error() string {
	return "Any operation that involves discovery is not allowed with the Let's Connect! client"
}

