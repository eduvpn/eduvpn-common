package eduvpn

import (
	"errors"
)

type VPNState struct {
	// Info passed by the client
	ConfigDirectory   string                       `json:"-"`
	Name              string                       `json:"-"`
	StateCallback     func(string, string, string) `json:"-"`
	StateCallbackData string                       `json:"-"`

	// The chosen server
	Servers Servers `json:"servers"`

	// The list of servers and organizations from disco
	DiscoList DiscoLists `json:"-"`

	// The file we keep open for logging
	LogFile FileLogger `json:"-"`

	// The fsm
	FSM FSM `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string), debug bool) error {
	if !state.InState(DEREGISTERED) {
		return errors.New("app already registered")
	}
	state.InitializeFSM()
	state.Name = name
	state.ConfigDirectory = directory
	state.StateCallback = stateCallback
	state.Debug = debug

	LogLevel := LOG_WARNING

	if debug {
		LogLevel = LOG_INFO
	}

	// Initialize the logger
	state.InitLog(LogLevel)

	// Try to load the previous configuration
	if state.LoadConfig() != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		state.Log(LOG_INFO, "Previous configuration not found")
	}
	state.GoTransition(NO_SERVER)
	return nil
}

func (state *VPNState) Deregister() error {
	// Close the log file
	state.CloseLog()

	// Write the config
	state.WriteConfig()

	// Re-initialize the servers and FSM
	state.Servers = Servers{}
	state.InitializeFSM()
	return nil
}

func (state *VPNState) Connect(url string) (string, error) {
	// New server chosen, ensure the server is fresh
	server := state.Servers.EnsureServer(url)
	// Make sure we are in the chosen state if available
	state.GoTransition(CHOSEN_SERVER)
	// Relogin with oauth
	// This moves the state to authenticated
	if server.NeedsRelogin() {
		loginErr := state.LoginOAuth()

		if loginErr != nil {
			return "", loginErr
		}
	} else { // OAuth was valid, ensure we are in the authenticated state
		state.GoTransition(AUTHENTICATED)
	}

	state.GoTransition(REQUEST_CONFIG)

	config, configErr := server.GetConfig()

	if configErr != nil {
		return "", configErr
	} else {
		state.GoTransition(HAS_CONFIG)
	}

	return config, nil
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}
