package eduvpn

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
}

func (state *VPNState) Register(name string, directory string, stateCallback func(string, string, string)) error {
	state.Name = name
	state.ConfigDirectory = directory
	state.StateCallback = stateCallback

	// Initialize the logger
	state.InitLog(LOG_WARNING)

	state.Log(LOG_INFO, "App registered")

	state.StateCallback("Start", "Registered", "app registered")

	// Try to load the previous configuration
	if state.LoadConfig() != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		state.Log(LOG_INFO, "Previous configuration not found")
	}
	return nil
}

func (state *VPNState) Deregister() {
	// Close the log file
	state.CloseLog()

	// Re-initialize everything
	state = &VPNState{}
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

	return state.Server.GetConfig()
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}
