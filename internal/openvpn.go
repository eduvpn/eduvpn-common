package internal

import "fmt"

func (server *Server) OpenVPNGetConfig() (string, error) {
	profile_id := server.Profiles.Current
	configOpenVPN, _, configErr := server.APIConnectOpenVPN(profile_id)

	if configErr != nil {
		return "", &OpenVPNGetConfigError{Err: configErr}
	}

	return configOpenVPN, nil
}

type OpenVPNGetConfigError struct {
	Err error
}

func (e *OpenVPNGetConfigError) Error() string {
	return fmt.Sprintf("failed getting OpenVPN config with error: %v", e.Err)
}
