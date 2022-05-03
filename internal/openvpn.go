package internal

import "fmt"

func OpenVPNGetConfig(server Server) (string, error) {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", &OpenVPNGetConfigError{Err: baseErr}
	}
	profile_id := base.Profiles.Current
	configOpenVPN, _, configErr := APIConnectOpenVPN(server, profile_id)

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
