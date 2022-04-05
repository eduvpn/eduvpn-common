package eduvpn

import (
	"errors"
)

type VPNState struct {
	// Info passed by the client
	ConfigDirectory string                       `json:"-"`
	Name            string                       `json:"-"`
	StateCallback   func(string, string, string) `json:"-"`

	// The chosen server
	Server *Server `json:"server"`

	// The list of servers and organizations from disco
	DiscoList *DiscoList `json:"disco"`

	// The file we keep open for logging
	LogFile *FileLogger `json:"-"`

	// The fsm
	FSM *FSM `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string), debug bool) error {
	state.InitializeFSM()
	if !state.HasTransition(APP_REGISTERED) {
		return errors.New("app already registered")
	}
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
	state.GoTransition(APP_REGISTERED, "HALLO")
	return nil
}

func (state *VPNState) Deregister() error {
	if !state.HasTransition(APP_DEREGISTERED) {
		return errors.New("app cannot deregister")
	}
	// Close the log file
	state.CloseLog()

	state.Server = &Server{}
	state.InitializeFSM()
	return nil
}

func (state *VPNState) Connect(url string) (string, error) {
	if state.Server == nil {
		state.Server = &Server{}
	}
	initializeErr := state.Server.Initialize(url)

	if initializeErr != nil {
		return "", initializeErr
	}

	if !state.Server.IsAuthenticated() {
		loginErr := state.LoginOAuth()

		if loginErr != nil {
			return "", loginErr
		}
	}

	config, configErr := state.Server.GetConfig()

	if configErr != nil {
		return "", configErr
	}

	if !state.HasTransition(SERVER_CONNECTED) {
		return "", errors.New("cannot connect to server, invalid state")
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
