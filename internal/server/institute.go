package server

import (
	"context"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

func (s *Servers) AddInstitute(ctx context.Context, disco *discovery.Discovery, id string, na bool) (*Server, error) {
	// This is basically done to double check if the server is part of the institute access section of disco
	dsrv, err := disco.ServerByURL(id, "institute_access")
	if err != nil {
		return nil, err
	}

	sd := api.ServerData{
		ID:         dsrv.BaseURL,
		Type:       server.TypeInstituteAccess,
		BaseWK:     dsrv.BaseURL,
		BaseAuthWK: dsrv.BaseURL,
	}

	var a *api.API
	if !na {
		// Authorize by creating the API object
		a, err = api.NewAPI(ctx, s.clientID, sd, s.cb, nil)
		if err != nil {
			return nil, err
		}
	}

	err = s.config.AddServer(dsrv.BaseURL, server.TypeInstituteAccess, v2.Server{LastAuthorizeTime: time.Now()})
	if err != nil {
		return nil, err
	}

	inst := s.NewServer(dsrv.BaseURL, server.TypeInstituteAccess, a)
	return &inst, nil
}

func (s *Servers) GetInstitute(ctx context.Context, id string, disco *discovery.Discovery, tok *eduoauth.Token, disableAuth bool) (*Server, error) {
	// This is basically done to double check if the server is part of the institute access section of disco
	dsrv, err := disco.ServerByURL(id, "institute_access")
	if err != nil {
		return nil, err
	}

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
