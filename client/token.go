package client

import (
	"errors"
	"fmt"

	srvtypes "codeberg.org/eduVPN/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

type cacheMap map[string]eduoauth.Token

// TokenCacher is a structure that caches tokens for each type of server
type TokenCacher struct {
	// InstituteAccess is the cached map for institute access servers
	InstituteAccess cacheMap
	// CustomServer is the cached map for custom server
	CustomServer cacheMap
	// SecureInternet is the cached map for the secure internet server
	SecureInternet *eduoauth.Token
}

// Get gets tokens from the cache map
func (c *cacheMap) Get(id string) (*eduoauth.Token, error) {
	if c == nil || len(*c) == 0 {
		return nil, errors.New("no cache map available")
	}
	if v, ok := (*c)[id]; ok {
		return &v, nil
	}
	return nil, fmt.Errorf("identifier: '%s' does not exist in token cache map", id)
}

// Get gets tokens using a server id and type from the cacher
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

// Set updates the cache for the server id `id` with tokens `t`
func (c *cacheMap) Set(id string, t eduoauth.Token) {
	if c == nil || len(*c) == 0 {
		*c = make(cacheMap)
	}
	(*c)[id] = t
}

// Set updates the top-level cacher for a specific server type
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
