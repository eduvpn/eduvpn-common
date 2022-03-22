package eduvpn

type VPNState struct {
	// Info passed by the client
	ConfigDirectory string `json:"-"`
	Name            string `json:"-"`

	// The chosen server
	Server *Server `json:"server"`

	// The list of servers and organizations from disco
	DiscoList *DiscoList `json:"disco"`
}

func Register(state *VPNState, name string, directory string, stateCallback func(string, string, string)) error {
	state.Name = name
	state.ConfigDirectory = directory

	stateCallback("START", "REGISTERED", "app registered")

	// Try to load the previous configuration

	if state.LoadConfig() != nil {
		// This error can be safely ignored, as when the config does not load, the struct will not be filled
		// Make sure to log this when we have implemented a good logging system
	}
	return nil
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}
