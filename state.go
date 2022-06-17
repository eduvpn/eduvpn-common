package eduvpn

import (
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/config"
	"github.com/jwijenbergh/eduvpn-common/internal/discovery"
	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/server"
)

type VPNState struct {
	// The chosen server
	Servers server.Servers `json:"servers"`

	// The list of servers and organizations from disco
	Discovery discovery.Discovery `json:"-"`

	// The fsm
	FSM fsm.FSM `json:"-"`

	// The logger
	Logger log.FileLogger `json:"-"`

	// The config
	Config config.Config `json:"-"`

	// Whether to enable debugging
	Debug bool `json:"-"`
}

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string), debug bool) error {
	if !state.FSM.InState(fsm.DEREGISTERED) {
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: fsm.DEREGISTERED}
	}
	// Initialize the logger
	logLevel := log.LOG_WARNING

	if debug {
		logLevel = log.LOG_INFO
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
		state.Logger.Log(log.LOG_INFO, "Previous configuration not found")
	}
	state.FSM.GoTransition(fsm.NO_SERVER)
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
	if !state.FSM.InState(fsm.OAUTH_STARTED) {
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: fsm.OAUTH_STARTED}
	}

	currentServer, serverErr := state.Servers.GetCurrentServer()

	if serverErr != nil {
		return &StateOAuthCancelError{Err: serverErr}
	}
	server.CancelOAuth(currentServer)
	return nil
}

func (state *VPNState) chooseServer(url string, isSecureInternet bool) (server.Server, error) {
	// New server chosen, ensure the server is fresh
	server, serverErr := state.Servers.EnsureServer(url, isSecureInternet, &state.FSM, &state.Logger)

	if serverErr != nil {
		return nil, serverErr
	}

	// Make sure we are in the chosen state if available
	state.FSM.GoTransition(fsm.CHOSEN_SERVER)
	return server, nil
}

func (state *VPNState) getConfigWithOptions(url string, isSecureInternet bool, forceTCP bool) (string, string, error) {
	if state.FSM.InState(fsm.DEREGISTERED) {
		return "", "", &StateFSMNotRegisteredError{}
	}

	// Go to no server if possible, else return an error
	if !state.FSM.InState(fsm.NO_SERVER) && !state.FSM.GoTransition(fsm.NO_SERVER) {
		return "", "", &fsm.FSMWrongStateTransitionError{Got: state.FSM.Current, Want: fsm.NO_SERVER}
	}

	// Make sure the server is chosen
	chosenServer, serverErr := state.chooseServer(url, isSecureInternet)

	if serverErr != nil {
		return "", "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: serverErr}
	}
	// Relogin with oauth
	// This moves the state to authorized
	if server.NeedsRelogin(chosenServer) {
		loginErr := server.Login(chosenServer)

		if loginErr != nil {
			// We are possibly in oauth started
			// So go to no server
			state.FSM.GoTransition(fsm.NO_SERVER)
			return "", "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: loginErr}
		}
	} else { // OAuth was valid, ensure we are in the authorized state
		state.FSM.GoTransition(fsm.AUTHORIZED)
	}

	state.FSM.GoTransition(fsm.REQUEST_CONFIG)

	config, configType, configErr := server.GetConfig(chosenServer, forceTCP)

	if configErr != nil {
		// Go back to no server if possible
		state.FSM.GoTransition(fsm.NO_SERVER)
		return "", "", &StateConnectError{URL: url, IsSecureInternet: isSecureInternet, Err: configErr}
	} else {
		state.FSM.GoTransition(fsm.HAS_CONFIG)
	}

	return config, configType, nil
}

func (state *VPNState) GetConfigInstituteAccess(url string, forceTCP bool) (string, string, error) {
	return state.getConfigWithOptions(url, false, forceTCP)
}

func (state *VPNState) GetConfigSecureInternet(url string, forceTCP bool) (string, string, error) {
	return state.getConfigWithOptions(url, true, forceTCP)
}

func (state *VPNState) GetDiscoOrganizations() (string, error) {
	if state.FSM.InState(fsm.DEREGISTERED) {
		return "", &StateWrongFSMStateError{Got: state.FSM.Current, Want: fsm.DEREGISTERED}
	}
	return state.Discovery.GetOrganizationsList()
}

func (state *VPNState) GetDiscoServers() (string, error) {
	if state.FSM.InState(fsm.DEREGISTERED) {
		return "", &StateFSMNotRegisteredError{}
	}
	return state.Discovery.GetServersList()
}

func (state *VPNState) SetProfileID(profileID string) error {
	if !state.FSM.InState(fsm.ASK_PROFILE) {
		return &StateWrongFSMStateError{Got: state.FSM.Current, Want: fsm.ASK_PROFILE}
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

func (state *VPNState) SetConnected() error {
	if !state.FSM.HasTransition(fsm.CONNECTED) {
		return &fsm.FSMWrongStateTransitionError{Got: state.FSM.Current, Want: fsm.CONNECTED}
	}

	state.FSM.GoTransition(fsm.CONNECTED)
	return nil
}

func (state *VPNState) SetDisconnected() error {
	if !state.FSM.HasTransition(fsm.HAS_CONFIG) {
		return &fsm.FSMWrongStateTransitionError{Got: state.FSM.Current, Want: fsm.HAS_CONFIG}
	}

	state.FSM.GoTransition(fsm.HAS_CONFIG)
	return nil
}

type StateSetProfileError struct {
	ProfileID string
	Err       error
}

func (e *StateSetProfileError) Error() string {
	return fmt.Sprintf("failed to set profile ID: %s with error: %v", e.ProfileID, e.Err)
}

type StateRegisterError struct {
	Err error
}

func (e *StateRegisterError) Error() string {
	return fmt.Sprintf("failed to register with error: %v", e.Err)
}

type StateFSMNotRegisteredError struct{}

func (e *StateFSMNotRegisteredError) Error() string {
	return fmt.Sprintf("state is not registered. Current FSM state: %s", fsm.DEREGISTERED.String())
}

type StateWrongFSMStateError struct {
	Got  fsm.FSMStateID
	Want fsm.FSMStateID
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
