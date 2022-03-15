package eduvpn

type EduVPNState struct {
	// The endpoints
	Endpoints *EduVPNEndpoints

	// Info passed by the client
	Name   string
	Server string

	// OAuth
	OAuthToken   *EduVPNOAuthToken
	OAuthSession *EduVPNOAuthSession
}

func Register(state *EduVPNState, name string, server string, stateCallback func(string, string)) error {
	state.Name = name
	state.Server = server

	endpoints, err := APIGetEndpoints(state)

	if err != nil {
		return err
	}

	state.Endpoints = endpoints
	stateCallback("START", "REGISTER")
	return nil
}

var VPNStateInstance *EduVPNState

func GetVPNState() *EduVPNState {
	if VPNStateInstance == nil {
		VPNStateInstance = &EduVPNState{}
	}
	return VPNStateInstance
}
