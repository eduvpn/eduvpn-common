package internal

func (server *Server) OpenVPNGetConfig() (string, error) {
	profile_id := server.Profiles.Current
	configOpenVPN, _, configErr := server.APIConnectOpenVPN(profile_id)

	if configErr != nil {
		return "", configErr
	}

	return configOpenVPN, nil
}
