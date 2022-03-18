package eduvpn

type VPNState struct {
	// Info passed by the client
	Name   string

	// The chosen server
	Server *Server
}

func Register(state *VPNState, name string, stateCallback func(string, string)) error {
	state.Name = name

	stateCallback("START", "REGISTER")
	return nil
}

var VPNStateInstance *VPNState

func GetVPNState() *VPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &VPNState{}
	}
	return VPNStateInstance
}
