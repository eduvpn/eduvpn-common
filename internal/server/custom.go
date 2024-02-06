package server

import (
	"context"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

func (s *Servers) AddCustom(ctx context.Context, id string, na bool) (*Server, error) {
	sd := api.ServerData{
		ID:         id,
		Type:       server.TypeCustom,
		BaseWK:     id,
		BaseAuthWK: id,
	}

	var a *api.API
	var err error
	if !na {
		// Authorize by creating the API object
		a, err = api.NewAPI(ctx, s.clientID, sd, s.cb, nil)
		if err != nil {
			return nil, err
		}
	}

	err = s.config.AddServer(id, server.TypeCustom, v2.Server{LastAuthorizeTime: time.Now()})
	if err != nil {
		return nil, err
	}

	cust := s.NewServer(id, server.TypeCustom, a)
	// Return the server with the API

	return &cust, nil
}

func (s *Servers) GetCustom(ctx context.Context, id string, tok *eduoauth.Token, disableAuth bool) (*Server, error) {
	sd := api.ServerData{
		ID:               id,
		Type:             server.TypeCustom,
		BaseWK:           id,
		BaseAuthWK:       id,
		DisableAuthorize: disableAuth,
	}

	// Get the server from the config
	_, err := s.config.GetServer(id, server.TypeCustom)
	if err != nil {
		return nil, err
	}
	a, err := api.NewAPI(ctx, s.clientID, sd, s.cb, tok)
	if err != nil {
		return nil, err
	}

	cust := s.NewServer(id, server.TypeCustom, a)
	return &cust, nil
}
