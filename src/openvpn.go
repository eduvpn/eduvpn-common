package eduvpn

func (server *Server) OpenVPNGetConfig() (string, error) {
	configOpenVPN, _, configErr := server.APIConnectOpenVPN("default")

	if configErr != nil {
		return "", configErr
	}

	return configOpenVPN, nil
}
