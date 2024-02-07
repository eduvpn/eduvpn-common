package client

import (
	"errors"
	"fmt"

	srvtypes "github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

type cacheMap map[string]eduoauth.Token

type TokenCacher struct {
	InstituteAccess cacheMap
	CustomServer    cacheMap
	SecureInternet  *eduoauth.Token
}

func (c *cacheMap) Get(id string) (*eduoauth.Token, error) {
	if c == nil || len(*c) == 0 {
		return nil, errors.New("no cache map available")
	}
	if v, ok := (*c)[id]; ok {
		return &v, nil
	}
	return nil, fmt.Errorf("identifier: '%s' does not exist in token cache map", id)
}

func (tc *TokenCacher) Get(id string, t srvtypes.Type) (*eduoauth.Token, error) {
	switch t {
	case srvtypes.TypeCustom:
		return tc.CustomServer.Get(id)
	case srvtypes.TypeInstituteAccess:
		return tc.InstituteAccess.Get(id)
	case srvtypes.TypeSecureInternet:
		if tc.SecureInternet == nil {
			return nil, errors.New("no secure internet server available")
		}
		return tc.SecureInternet, nil
	}
	return nil, fmt.Errorf("invalid type for token cacher get: %d", t)
}

func (c *cacheMap) Set(id string, t eduoauth.Token) {
	if c == nil || len(*c) == 0 {
		*c = make(cacheMap)
	}
	(*c)[id] = t
}

func (tc *TokenCacher) Set(id string, t srvtypes.Type, tok eduoauth.Token) error {
	switch t {
	case srvtypes.TypeCustom:
		tc.CustomServer.Set(id, tok)
		return nil
	case srvtypes.TypeInstituteAccess:
		tc.InstituteAccess.Set(id, tok)
		return nil
	case srvtypes.TypeSecureInternet:
		tc.SecureInternet = &tok
		return nil
	}
	return fmt.Errorf("invalid type for token cacher set: %d", t)
}
