package server

import (
	"context"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

// AddCustom adds a custom server to the internal server list
// `ctx` is the context used for cancellation
// `id` is the identifier of the server, the base URL
// `ot` specifies specifies the start time OAuth was already triggered
func (s *Servers) AddCustom(ctx context.Context, id string, ot *int64) error {
	sd := api.ServerData{
		ID:         id,
		Type:       server.TypeCustom,
		BaseWK:     id,
		BaseAuthWK: id,
	}

	auth := time.Time{}
	if ot != nil {
		auth = time.Unix(*ot, 0)
	}
	err := s.config.AddServer(id, server.TypeCustom, v2.Server{
		LastAuthorizeTime: auth,
	})
	if err != nil {
		return err
	}

	// no authorization should be triggered, return
	if ot != nil {
		return nil
	}

	// Authorize by creating the API object
	_, err = api.NewAPI(ctx, s.clientID, sd, s.cb, nil)
	if err != nil {
		// authorization has failed, remove the server again
		rerr := s.config.RemoveServer(id, server.TypeCustom)
		if rerr != nil {
			log.Logger.Warningf("could not remove custom server: '%s' after failing authorization: %v", id, rerr)
		}
		return err
	}
	return nil
}

// GetCustom gets a custom server
// `ctx` is the context for cancellation
// `id` is the identifier of the server
// `tok` are the tokens such that we can initialize the API
// `disableAuth` is set to True when authorization should not be triggered
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
