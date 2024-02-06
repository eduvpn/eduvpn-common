package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	srvtypes "github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

type Callbacks interface {
	api.Callbacks
	GettingConfig() error
	InvalidProfile(context.Context, *Server) (string, error)
}

type Servers struct {
	clientID  string
	cb        Callbacks
	WGSupport bool
	config    *v2.V2
}

func (s *Servers) Remove(identifier string, t srvtypes.Type) error {
	return s.config.RemoveServer(identifier, t)
}

func NewServers(name string, cb Callbacks, wgSupport bool, cfg *v2.V2) Servers {
	return Servers{
		clientID:  name,
		cb:        cb,
		WGSupport: wgSupport,
		config:    cfg,
	}
}

type CurrentServer struct {
	*v2.Server
	T    v2.ServerType
	srvs *Servers
}

func (cs *CurrentServer) ServerWithCallbacks(ctx context.Context, disco *discovery.Discovery, tokens *eduoauth.Token, disableAuth bool) (*Server, error) {
	switch cs.T.T {
	case srvtypes.TypeInstituteAccess:
		return cs.srvs.GetInstitute(ctx, cs.T.ID, disco, tokens, disableAuth)
	case srvtypes.TypeSecureInternet:
		return cs.srvs.GetSecure(ctx, cs.T.ID, disco, tokens, disableAuth)
	case srvtypes.TypeCustom:
		return cs.srvs.GetCustom(ctx, cs.T.ID, tokens, disableAuth)
	default:
		return nil, fmt.Errorf("no such server type: %d", cs.T.T)
	}
}

func (s *Servers) GetServer(id string, t srvtypes.Type) (*v2.Server, error) {
	if s.config == nil {
		return nil, errors.New("no configuration available")
	}
	return s.config.GetServer(id, t)
}

func (s *Servers) CurrentServer() (*CurrentServer, error) {
	curr, k, err := s.config.CurrentServer()
	if err != nil {
		return nil, err
	}
	return &CurrentServer{
		Server: curr,
		T:      *k,
		srvs:   s,
	}, nil
}

func (s *Servers) PublicCurrent(disco *discovery.Discovery) (*srvtypes.Current, error) {
	return s.config.PublicCurrent(disco)
}

func (s *Servers) ConnectWithCallbacks(ctx context.Context, srv *Server, pTCP bool) (*srvtypes.Configuration, error) {
	err := srv.SetCurrent()
	if err != nil {
		return nil, err
	}
	err = s.cb.GettingConfig()
	if err != nil {
		return nil, err
	}
	cfg, err := srv.connect(ctx, s.WGSupport, pTCP)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, ErrInvalidProfile) {
		return cfg, err
	}
	// Get a new profile from the callback
	pr, err := s.cb.InvalidProfile(ctx, srv)
	if err != nil {
		return cfg, err
	}
	err = srv.SetProfileID(pr)
	if err != nil {
		return nil, err
	}
	err = s.cb.GettingConfig()
	if err != nil {
		return nil, err
	}
	return srv.connect(ctx, s.WGSupport, pTCP)
}
