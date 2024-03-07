// Package v2 implements version 2 of the state file
package v2

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/types/server"
)

// Server is the struct for each server
type Server struct {
	// Profiles are the list of profiles
	Profiles server.Profiles `json:"profiles"`
	// LastAuthorizeTime is the time we last authorized
	// This is used for determining when to show e.g. the renew button
	LastAuthorizeTime time.Time `json:"last_authorize_time,omitempty"`
	// ExpireTime is the time at which the VPN expires
	ExpireTime time.Time `json:"expire_time,omitempty"`

	// CountryCode is the country code for the server in case of secure internet
	// Otherwise it is an empty string
	CountryCode string `json:"country_code,omitempty"`
}

// ServerKey is the key type of the server map
type ServerKey struct {
	// T is the type of server, e.g. secure internet
	T server.Type
	// ID is the identifier for the server
	ID string
}

const keyFormat = "%d,%s"

func newServerType(key string) (*ServerKey, error) {
	var t server.Type
	var id string
	if _, err := fmt.Sscanf(key, keyFormat, &t, &id); err != nil {
		return nil, err
	}

	return &ServerKey{
		T:  t,
		ID: id,
	}, nil
}

// MarshalText convers the server key into one that can be used in a map
func (st ServerKey) MarshalText() ([]byte, error) {
	k := fmt.Sprintf(keyFormat, st.T, st.ID)
	return []byte(k), nil
}

// UnmarshalText converts the marshaled key into a ServerType struct
func (st *ServerKey) UnmarshalText(text []byte) error {
	k := string(text)
	g, err := newServerType(k)
	if err != nil {
		return err
	}
	*st = *g
	return nil
}

// V2 is the top-level struct for the state file
type V2 struct {
	// List is the list of servers
	List map[ServerKey]*Server `json:"server_list,omitempty"`
	// LastChosen represents the key of the last chosen server
	// A server is chosen if we got a config for it
	LastChosen *ServerKey `json:"last_chosen_id,omitempty"`
	// Discovery is the cached list of discovery JSON
	Discovery discovery.Discovery `json:"discovery"`
}

// RemoveServer removes a server with id `id` and type `t` from the V2 struct
// It returns an error if no such server exists
func (cfg *V2) RemoveServer(id string, t server.Type) error {
	k := ServerKey{
		ID: id,
		T:  t,
	}

	if _, ok := cfg.List[k]; ok {
		delete(cfg.List, k)

		// reset the last chosen
		if cfg.LastChosen != nil && *cfg.LastChosen == k {
			cfg.LastChosen = nil
		}
		return nil
	}
	return errors.New("server does not exist")
}

func (cfg *V2) getServerWithKey(k ServerKey) (*Server, error) {
	if v, ok := cfg.List[k]; ok {
		return v, nil
	}
	return nil, errors.New("server does not exist")
}

// GetServer gets a server with id `id` and type `t`
// If the server doesn't exist it returns nil and an error
func (cfg *V2) GetServer(id string, t server.Type) (*Server, error) {
	k := ServerKey{
		ID: id,
		T:  t,
	}
	return cfg.getServerWithKey(k)
}

// CurrentServer gets the last chosen server
// It returns the server, the server type and an error if it doesn't exist
func (cfg *V2) CurrentServer() (*Server, *ServerKey, error) {
	if cfg.LastChosen == nil {
		return nil, nil, errors.New("no server chosen before")
	}
	srv, err := cfg.getServerWithKey(*cfg.LastChosen)
	if err != nil {
		return nil, nil, err
	}
	return srv, cfg.LastChosen, nil
}

// HasSecureInternet returns true whether or not the state file
// has a secure internet server in it
func (cfg *V2) HasSecureInternet() bool {
	for k := range cfg.List {
		if k.T == server.TypeSecureInternet {
			return true
		}
	}
	return false
}

// AddServer adds a server with id `id`, type `t` and server `srv`
func (cfg *V2) AddServer(id string, t server.Type, srv Server) error {
	if cfg.HasSecureInternet() && t == server.TypeSecureInternet {
		return errors.New("a secure internet server already exists, remove the other secure internet server first")
	}
	k := ServerKey{
		ID: id,
		T:  t,
	}
	if cfg.List == nil {
		cfg.List = make(map[ServerKey]*Server)
	}
	cfg.List[k] = &srv
	return nil
}

// PublicCurrent gets the current server as a type that should be returned to the client
// It returns this server or nil and an error if it doesn't exist
func (cfg *V2) PublicCurrent(disco *discovery.Discovery) (*server.Current, error) {
	curr, _, err := cfg.CurrentServer()
	if err != nil {
		return nil, err
	}
	rcurr := &server.Current{}
	// SAFETY: LastChosen is guaranteed to be non-nil here
	switch cfg.LastChosen.T {
	case server.TypeInstituteAccess:
		g, err := convertInstitute(cfg.LastChosen.ID, disco)
		if err != nil {
			return nil, err
		}
		g.Profiles = curr.Profiles
		rcurr.Institute = g
	case server.TypeSecureInternet:
		g, err := convertSecure(cfg.LastChosen.ID, curr.CountryCode, disco)
		if err != nil {
			return nil, err
		}
		g.Profiles = curr.Profiles
		rcurr.SecureInternet = g
	case server.TypeCustom:
		g, err := convertCustom(cfg.LastChosen.ID)
		if err != nil {
			return nil, err
		}
		g.Profiles = curr.Profiles
		rcurr.Custom = g
	default:
		return nil, fmt.Errorf("unknown connected type: %d", cfg.LastChosen.T)
	}
	rcurr.Type = cfg.LastChosen.T
	return rcurr, nil
}

func convertInstitute(url string, disco *discovery.Discovery) (*server.Institute, error) {
	dsrv, err := disco.ServerByURL(url, "institute_access")
	if err != nil {
		return nil, err
	}

	return &server.Institute{
		Server: server.Server{
			DisplayName: dsrv.DisplayName,
			Identifier:  url,
		},
		SupportContacts: dsrv.SupportContact,
	}, nil
}

func convertCustom(u string) (*server.Server, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	return &server.Server{
		DisplayName: map[string]string{
			"en": pu.Hostname(),
		},
		Identifier: u,
	}, nil
}

func convertSecure(orgID string, countryCode string, disco *discovery.Discovery) (*server.SecureInternet, error) {
	dorg, _, err := disco.SecureHomeArgs(orgID)
	if err != nil {
		return nil, err
	}
	return &server.SecureInternet{
		Server: server.Server{
			DisplayName: dorg.DisplayName,
			Identifier:  dorg.OrgID,
		},
		CountryCode: countryCode,
		Locations:   disco.SecureLocationList(),
	}, nil
}

// PublicList gets all the servers in a format that is returned to the client
func (cfg *V2) PublicList(disco *discovery.Discovery) *server.List {
	ret := &server.List{}
	// TODO: profile information?
	for k, v := range cfg.List {
		switch k.T {
		case server.TypeInstituteAccess:
			g, err := convertInstitute(k.ID, disco)
			if err != nil || g == nil {
				// TODO: log/delisted?
				continue
			}
			g.Profiles = v.Profiles
			ret.Institutes = append(ret.Institutes, *g)
		case server.TypeSecureInternet:
			g, err := convertSecure(k.ID, v.CountryCode, disco)
			if err != nil || g == nil {
				// TODO: log/delisted?
				continue
			}
			g.Profiles = v.Profiles
			ret.SecureInternet = g
		case server.TypeCustom:
			g, err := convertCustom(k.ID)
			if err != nil || g == nil {
				// TODO: log/delisted?
				continue
			}
			g.Profiles = v.Profiles
			ret.Custom = append(ret.Custom, *g)
		default:
			// TODO: log
			continue
		}
	}
	return ret
}
