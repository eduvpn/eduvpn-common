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
		return &types.WrappedErrorMessage{Message: errorMessage, Err: baseErr}
	}

	if base.Type != "custom_server" {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: errors.New("Not a custom server")}
	}

	if _, ok := servers.CustomServers.Map[base.URL]; ok {
		servers.CustomServers.CurrentURL = base.URL
		servers.IsType = CustomServerType
	} else {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: errors.New("Not a custom server")}
	}
	return nil
}

func (servers *Servers) GetCustomServer(url string) (*InstituteAccessServer, error) {
	if server, ok := servers.CustomServers.Map[url]; ok {
		return server, nil
	}
	return nil, &types.WrappedErrorMessage{Message: "failed to get institute access server", Err: fmt.Errorf("No custom server with URL: %s", url)}
}

func (servers *Servers) RemoveCustomServer(url string) {
	servers.CustomServers.Remove(url)
}
