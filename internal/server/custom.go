package server

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/types"
)

func (servers *Servers) SetCustomServer(server Server) error {
	errorMessage := "failed setting custom server"
	base, baseErr := server.GetBase()
	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	if base.Type != "custom_server" {
		return types.NewWrappedError(errorMessage, errors.New("not a custom server"))
	}

	if _, ok := servers.CustomServers.Map[base.URL]; ok {
		servers.CustomServers.CurrentURL = base.URL
		servers.IsType = CustomServerType
	} else {
		return types.NewWrappedError(errorMessage, errors.New("not a custom server"))
	}
	return nil
}

func (servers *Servers) GetCustomServer(url string) (*InstituteAccessServer, error) {
	if server, ok := servers.CustomServers.Map[url]; ok {
		return server, nil
	}
	return nil, types.NewWrappedError("failed to get institute access server", fmt.Errorf("no custom server with URL: %s", url))
}

func (servers *Servers) RemoveCustomServer(url string) {
	servers.CustomServers.Remove(url)
}
