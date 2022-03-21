package eduvpn

type VPNState struct {
	// Info passed by the client
	ConfigDirectory string `json:"-"`
	Name            string `json:"-"`

	// The chosen server
	Server *Server `json:"server"`
}

func Register(state *VPNState, name string, directory string, stateCallback func(string, string, string)) error {
	state.Name = name
	state.ConfigDirectory = directory

	stateCallback("START", "REGISTERED", "test data")
	return nil
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}
