package server

import (
	"context"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

// AddInstitute adds an institute access server
// `ctx` is the context used for cancellation
// `disco` are the discovery servers
// `id` is the identifier for the server, the base url
// `ot` specifies specifies the start time OAuth was already triggered
func (s *Servers) AddInstitute(ctx context.Context, discom *discovery.Manager, id string, ot *int64) error {
	// This is basically done to double check if the server is part of the institute access section of disco
	disco, release := discom.Discovery(false)
	dsrv, err := disco.ServerByURL(id, "institute_access")
	if err != nil {
		release()
		return err
	}
	release()

	sd := api.ServerData{
		ID:         dsrv.BaseURL,
		Type:       server.TypeInstituteAccess,
		BaseWK:     dsrv.BaseURL,
		BaseAuthWK: dsrv.BaseURL,
	}

	auth := time.Time{}
	if ot != nil {
		auth = time.Unix(*ot, 0)
	}

	err = s.config.AddServer(dsrv.BaseURL, server.TypeInstituteAccess, v2.Server{
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
		rerr := s.config.RemoveServer(dsrv.BaseURL, server.TypeInstituteAccess)
		if rerr != nil {
			log.Logger.Warningf("could not remove institute access server: '%s' after failing authorization: %v", dsrv.BaseURL, rerr)
		}
		return err
	}
	return nil
}

// GetInstitute gets an institute access server
// `ctx` is the context used for cancellation
// `id` is the identifier for the server, the base url
// `disco` are the discovery servers
// `tok` are the tokens such that we do not have to trigger auth
// `disableAuth` is true when auth should never be triggered
func (s *Servers) GetInstitute(ctx context.Context, id string, discom *discovery.Manager, tok *eduoauth.Token, disableAuth bool) (*Server, error) {
	disco, release := discom.Discovery(false)
	// This is basically done to double check if the server is part of the institute access section of disco
	dsrv, err := disco.ServerByURL(id, "institute_access")
	if err != nil {
		release()
		return nil, err
	}
	release()

	// Get the server from the config
	_, err = s.config.GetServer(dsrv.BaseURL, server.TypeInstituteAccess)
	if err != nil {
		return nil, err
	}
	sd := api.ServerData{
		ID:               dsrv.BaseURL,
		Type:             server.TypeInstituteAccess,
		BaseWK:           dsrv.BaseURL,
		BaseAuthWK:       dsrv.BaseURL,
		DisableAuthorize: disableAuth,
	}
	// Authorize by creating the API object
	a, err := api.NewAPI(ctx, s.clientID, sd, s.cb, tok)
	if err != nil {
		return nil, err
	}

	inst := s.NewServer(dsrv.BaseURL, server.TypeInstituteAccess, a)
	return &inst, nil
}
