package eduvpn

import (
	"errors"
	"github.com/jwijenbergh/eduvpn-common/internal"
)

type VPNState struct {
	// The chosen server
	Servers internal.Servers `json:"servers"`

	// The list of servers and organizations from disco
	Discovery internal.Discovery `json:"-"`

	// The fsm
	FSM internal.FSM `json:"-"`

	// The logger
	Logger internal.FileLogger `json:"-"`

	// The config
	Config internal.Config `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string), debug bool) error {
	if !state.FSM.InState(internal.DEREGISTERED) {
		return errors.New("app already registered")
	}
	// Initialize the logger
	logLevel := internal.LOG_WARNING

	if debug {
		logLevel = internal.LOG_INFO
	}

	loggerErr := state.Logger.Init(logLevel, name, directory)
	if loggerErr != nil {
		return errors.New("Failed to create a logger")
	}

	// Initialize the FSM
	state.FSM.Init(name, stateCallback, &state.Logger, directory, debug)
	state.Debug = debug

	// Initialize the Config
	state.Config.Init(name, directory)

	// Initialize Discovery
	state.Discovery.Init(&state.FSM, &state.Logger)

	// Try to load the previous configuration
	if state.Config.Load(&state) != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		state.Logger.Log(internal.LOG_INFO, "Previous configuration not found")
	}
	state.FSM.GoTransition(internal.NO_SERVER)
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

func (state *VPNState) CancelOAuth() error {
	if !state.FSM.InState(internal.OAUTH_STARTED) {
		return errors.New("cannot cancel oauth, oauth not started")
	}

	server, serverErr := state.Servers.GetCurrentServer()

	if serverErr != nil {
		return serverErr
	}
	server.CancelOAuth()
	return nil
}

func (state *VPNState) Connect(url string) (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", errors.New("app not registered")
	}
	// New server chosen, ensure the server is fresh
	server, serverErr := state.Servers.EnsureServer(url, &state.FSM, &state.Logger)

	if serverErr != nil {
		return "", serverErr
	}
	// Make sure we are in the chosen state if available
	state.FSM.GoTransition(internal.CHOSEN_SERVER)
	// Relogin with oauth
	// This moves the state to authorized
	if server.NeedsRelogin() {
		loginErr := server.Login()

		if loginErr != nil {
			// We are possibly in oauth started
			// So go to chosen server
			state.FSM.GoTransition(internal.CHOSEN_SERVER)
			return "", loginErr
		}
	} else { // OAuth was valid, ensure we are in the authorized state
		state.FSM.GoTransition(internal.AUTHORIZED)
	}

	state.FSM.GoTransition(internal.REQUEST_CONFIG)

	config, configErr := server.GetConfig()

	if configErr != nil {
		return "", configErr
	} else {
		state.FSM.GoTransition(internal.HAS_CONFIG)
	}

	return config, nil
}

func (state *VPNState) GetDiscoOrganizations() (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", errors.New("app not registered")
	}
	return state.Discovery.GetOrganizationsList()
}


func (state *VPNState) GetDiscoServers() (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", errors.New("app not registered")
	}
	return state.Discovery.GetServersList()
}

func (state *VPNState) SetProfileID(profileID string) error {
	if !state.FSM.InState(internal.ASK_PROFILE) {
		return errors.New("Invalid state for setting a profile")
	}

	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		return errors.New("No server found for setting a profile ID")
	}
	server.Profiles.Current = profileID
	return nil
}
