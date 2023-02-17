package server

import (
	"github.com/go-errors/errors"
)

func (ss *Servers) SetCustomServer(server Server) error {
	b, err := server.Base()
	if err != nil {
		return err
	}

	if b.Type != "custom_server" {
		return errors.WrapPrefix(err, "not a custom server", 0)
	}

	if _, ok := ss.CustomServers.Map[b.URL]; ok {
		ss.CustomServers.CurrentURL = b.URL
		ss.IsType = CustomServerType
	} else {
		return errors.Errorf("this server is not yet added as a custom server: %s", b.URL)
	}
	return nil
}

func (ss *Servers) GetCustomServer(url string) (*InstituteAccessServer, error) {
	if srv, ok := ss.CustomServers.Map[url]; ok {
		return srv, nil
	}
	return nil, errors.Errorf("failed to get institute access server - no custom server with URL '%s'", url)
}

func (ss *Servers) RemoveCustomServer(url string) {
	ss.CustomServers.Remove(url)
}
