package v2

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/types/server"
)

type Server struct {
	Profiles          server.Profiles `json:"profiles"`
	LastAuthorizeTime time.Time       `json:"last_authorize_time,omitempty"`
	ExpireTime        time.Time       `json:"expire_time,omitempty"`

	// In case of secure internet:
	CountryCode string `json:"country_code"`
}

type ServerType struct {
	T  server.Type
	ID string
}

const keyFormat = "%d,%s"

func newServerType(key string) (*ServerType, error) {
	var t server.Type
	var id string
	if _, err := fmt.Sscanf(key, keyFormat, &t, &id); err != nil {
		return nil, err
	}

	return &ServerType{
		T:  t,
		ID: id,
	}, nil
}

func (st ServerType) MarshalText() ([]byte, error) {
	k := fmt.Sprintf(keyFormat, st.T, st.ID)
	return []byte(k), nil
}

func (st *ServerType) UnmarshalText(text []byte) error {
	k := string(text)
	g, err := newServerType(k)
	if err != nil {
		return err
	}
	*st = *g
	return nil
}

type V2 struct {
	List       map[ServerType]*Server `json:"server_list,omitempty"`
	LastChosen *ServerType            `json:"last_chosen_id,omitempty"`
	Discovery  discovery.Discovery    `json:"discovery"`
}

func (cfg *V2) RemoveServer(id string, t server.Type) error {
	k := ServerType{
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

func (cfg *V2) getServerWithKey(k ServerType) (*Server, error) {
	if v, ok := cfg.List[k]; ok {
		return v, nil
	}
	return nil, errors.New("server does not exist")
}

func (cfg *V2) GetServer(id string, t server.Type) (*Server, error) {
	k := ServerType{
		ID: id,
		T:  t,
	}
	return cfg.getServerWithKey(k)
}

func (cfg *V2) CurrentServer() (*Server, *ServerType, error) {
	if cfg.LastChosen == nil {
		return nil, nil, errors.New("no server chosen before")
	}
	srv, err := cfg.getServerWithKey(*cfg.LastChosen)
	if err != nil {
		return nil, nil, err
	}
	return srv, cfg.LastChosen, nil
}

func (cfg *V2) HasSecureInternet() bool {
	for k := range cfg.List {
		if k.T == server.TypeSecureInternet {
			return true
		}
	}
	return false
}

func (cfg *V2) AddServer(id string, t server.Type, srv Server) error {
	if cfg.HasSecureInternet() && t == server.TypeSecureInternet {
		return errors.New("a secure internet server already exists, remove the other secure internet server first")
	}
	k := ServerType{
		ID: id,
		T:  t,
	}
	if cfg.List == nil {
		cfg.List = make(map[ServerType]*Server)
	}
	cfg.List[k] = &srv
	return nil
}

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
