package server

import (
	"fmt"
	"github.com/eduvpn/eduvpn-common/types"
)

func (servers *Servers) GetCustomServer(url string) (*InstituteAccessServer, error) {
	if server, ok := servers.CustomServers.Map[url]; ok {
		return server, nil
	}
	return nil, &types.WrappedErrorMessage{Message: "failed to get institute access server", Err: fmt.Errorf("No custom server with URL: %s", url)}
}

func (servers *Servers) RemoveCustomServer(url string) {
	servers.CustomServers.Remove(url)
}
