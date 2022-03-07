package eduvpn

type EduVPNState struct {
	// The struct used for oauth
	OAuth *EduVPNOauth

	// The endpoints
	Endpoints *EduVPNEndpoints

	// Info passed by the client
	Name   string
	Server string
}

func Register(state *EduVPNState, name string, server string) error {
	state.Name = name
	state.Server = server

	endpoints, err := APIGetEndpoints(state)

	if err != nil {
		return err
	}

	state.Endpoints = endpoints
	return nil
}


var VPNStateInstance *EduVPNState

func GetVPNState() *EduVPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &EduVPNState{}
	}
	return VPNStateInstance
}


