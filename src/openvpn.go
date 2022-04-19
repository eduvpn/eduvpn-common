package eduvpn

func (server *Server) OpenVPNGetConfig(profile_id string) (string, error) {
	configOpenVPN, _, configErr := server.APIConnectOpenVPN(profile_id)

	if configErr != nil {
		return "", configErr
	}

	return configOpenVPN, nil
}
