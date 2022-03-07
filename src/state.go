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

func Register(name string, server string) *EduVPNState {
	state := &EduVPNState{Name: name, Server: server}
	endpoints, err := APIGetEndpoints(state)

	if err != nil {
		panic(err)
	}
	state.Endpoints = endpoints
	return state
}
