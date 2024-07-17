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

// Callbacks defines the interface for doing certain callback operations
type Callbacks interface {
	// api.Callbacks is the API callback interface
	api.Callbacks
	// GettingConfig is called when the config is obtained
	GettingConfig() error
	// InvalidProfile is called when an invalid profile is found
	InvalidProfile(context.Context, *Server) (string, error)
}

// Servers is the main struct that contains information for configuring the servers
type Servers struct {
	clientID string
	cb       Callbacks
	config   *v2.V2
}

// Remove removes a server with id `identifier` and type `t`
func (s *Servers) Remove(identifier string, t srvtypes.Type) error {
	return s.config.RemoveServer(identifier, t)
}

// NewServers creates a new servers struct
func NewServers(name string, cb Callbacks, cfg *v2.V2) Servers {
	return Servers{
		clientID: name,
		cb:       cb,
		config:   cfg,
	}
}

// CurrentServer contains the information for the current active server
type CurrentServer struct {
	// it embeds the state file server
	*v2.Server
	// Key is the server key
	Key v2.ServerKey
	// srvs refers to the original servers manager
	srvs *Servers
}

// ServerWithCallbacks gets the current server as a server struct and triggers callbacks as needed
func (cs *CurrentServer) ServerWithCallbacks(ctx context.Context, discom *discovery.Manager, tokens *eduoauth.Token, disableAuth bool) (*Server, error) {
	switch cs.Key.T {
	case srvtypes.TypeInstituteAccess:
		return cs.srvs.GetInstitute(ctx, cs.Key.ID, discom, tokens, disableAuth)
	case srvtypes.TypeSecureInternet:
		return cs.srvs.GetSecure(ctx, cs.Key.ID, discom, tokens, disableAuth)
	case srvtypes.TypeCustom:
		return cs.srvs.GetCustom(ctx, cs.Key.ID, tokens, disableAuth)
	default:
		return nil, fmt.Errorf("no such server type: %d", cs.Key.T)
	}
}

// GetServer gets a server from the state file
func (s *Servers) GetServer(id string, t srvtypes.Type) (*v2.Server, error) {
	if s.config == nil {
		return nil, errors.New("no configuration available")
	}
	return s.config.GetServer(id, t)
}

// CurrentServer gets the current server from the state file and wraps it into a neat type
func (s *Servers) CurrentServer() (*CurrentServer, error) {
	curr, k, err := s.config.CurrentServer()
	if err != nil {
		return nil, err
	}
	return &CurrentServer{
		Server: curr,
		Key:    *k,
		srvs:   s,
	}, nil
}

// PublicCurrent gets the current server into a type that we can return to the client
func (s *Servers) PublicCurrent(discom *discovery.Manager) (*srvtypes.Current, error) {
	disco, release := discom.Discovery(false)
	defer release()
	return s.config.PublicCurrent(disco)
}

// ConnectWithCallbacks handles the /connect flow
// It calls callbacks as needed
func (s *Servers) ConnectWithCallbacks(ctx context.Context, srv *Server, pTCP bool) (*srvtypes.Configuration, error) {
	err := srv.SetCurrent()
	if err != nil {
		return nil, err
	}
	err = s.cb.GettingConfig()
	if err != nil {
		return nil, err
	}
	cfg, err := srv.connect(ctx, pTCP)
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
	return srv.connect(ctx, pTCP)
}
