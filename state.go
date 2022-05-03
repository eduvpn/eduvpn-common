package eduvpn

import (
	"fmt"

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

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string), debug bool) error {
	if !state.FSM.InState(internal.DEREGISTERED) {
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: internal.DEREGISTERED}
	}
	// Initialize the logger
	logLevel := internal.LOG_WARNING

	if debug {
		logLevel = internal.LOG_INFO
	}

	loggerErr := state.Logger.Init(logLevel, name, directory)
	if loggerErr != nil {
		return &StateRegisterError{Err: loggerErr}
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
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: internal.OAUTH_STARTED}
	}

	server, serverErr := state.Servers.GetCurrentServer()

	if serverErr != nil {
		return &StateOAuthCancelError{Err: serverErr}
	}
	internal.CancelOAuth(server)
	return nil
}

func (state *VPNState) chooseServer(url string, isSecureInternet bool) (internal.Server, error) {
	// New server chosen, ensure the server is fresh
	server, serverErr := state.Servers.EnsureServer(url, isSecureInternet, &state.FSM, &state.Logger)

	if serverErr != nil {
		return nil, serverErr
	}

	// Make sure we are in the chosen state if available
	state.FSM.GoTransition(internal.CHOSEN_SERVER)
	return server, nil
}

func (state *VPNState) connectWithOptions(url string, isSecureInternet bool) (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", &StateFSMNotRegisteredError{}
	}

	// Make sure the server is chosen
	server, serverErr := state.chooseServer(url, isSecureInternet)

	if serverErr != nil {
		return "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: serverErr}
	}
	// Relogin with oauth
	// This moves the state to authorized
	if internal.NeedsRelogin(server) {
		loginErr := internal.Login(server)

		if loginErr != nil {
			// We are possibly in oauth started
			// So go to chosen server
			state.FSM.GoTransition(internal.CHOSEN_SERVER)
			return "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: loginErr}
		}
	} else { // OAuth was valid, ensure we are in the authorized state
		state.FSM.GoTransition(internal.AUTHORIZED)
	}

	state.FSM.GoTransition(internal.REQUEST_CONFIG)

	config, configErr := internal.GetConfig(server)

	if configErr != nil {
		return "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: configErr}
	} else {
		state.FSM.GoTransition(internal.HAS_CONFIG)
	}

	return config, nil
}

func (state *VPNState) ConnectInstituteAccess(url string) (string, error) {
	return state.connectWithOptions(url, false)
}

func (state *VPNState) ConnectSecureInternet(url string) (string, error) {
	return state.connectWithOptions(url, true)
}

func (state *VPNState) GetDiscoOrganizations() (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", &StateWrongFSMStateError{Got: state.FSM.Current, Want: internal.DEREGISTERED}
	}
	return state.Discovery.GetOrganizationsList()
}

func (state *VPNState) GetDiscoServers() (string, error) {
	if state.FSM.InState(internal.DEREGISTERED) {
		return "", &StateFSMNotRegisteredError{}
	}
	return state.Discovery.GetServersList()
}

func (state *VPNState) SetProfileID(profileID string) error {
	if !state.FSM.InState(internal.ASK_PROFILE) {
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: internal.ASK_PROFILE}
	}

	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		return &StateSetProfileError{ProfileID: profileID, Err: serverErr}
	}

	base, baseErr := server.GetBase()

	if baseErr != nil {
		return &StateSetProfileError{ProfileID: profileID, Err: baseErr}
	}
	base.Profiles.Current = profileID
	return nil
}

type StateSetProfileError struct {
	ProfileID string
	Err       error
}

func (e *StateSetProfileError) Error() string {
	return fmt.Sprintf("failed to set profile ID %s with error %v", e.ProfileID, e.Err)
}

type StateRegisterError struct {
	Err error
}

func (e *StateRegisterError) Error() string {
	return fmt.Sprintf("failed to register with error %v", e.Err)
}

type StateFSMNotRegisteredError struct{}

func (e *StateFSMNotRegisteredError) Error() string {
	return fmt.Sprintf("state is not registered. Current FSM state: %s", internal.DEREGISTERED.String())
}

type StateWrongFSMStateError struct {
	Got  internal.FSMStateID
	Want internal.FSMStateID
}

func (e *StateWrongFSMStateError) Error() string {
	return fmt.Sprintf("wrong FSM state, got: %s, want: %s", e.Got.String(), e.Want.String())
}

type StateOAuthCancelError struct {
	Err error
}

func (e *StateOAuthCancelError) Error() string {
	return fmt.Sprintf("failed cancelling OAuth for state with error: %v", e.Err)
}

type StateConnectError struct {
	URL              string
	IsSecureInternet bool
	Err              error
}

func (e *StateConnectError) Error() string {
	return fmt.Sprintf("failed connecting to server: %s (is secure internet: %v) with error: %v", e.URL, e.IsSecureInternet, e.Err)
}
