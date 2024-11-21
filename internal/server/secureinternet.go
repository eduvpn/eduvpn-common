package server

import (
	"context"
	"errors"
	"time"

	"codeberg.org/eduVPN/eduvpn-common/internal/api"
	"codeberg.org/eduVPN/eduvpn-common/internal/config/v2"
	"codeberg.org/eduVPN/eduvpn-common/internal/discovery"
	"codeberg.org/eduVPN/eduvpn-common/internal/log"
	"codeberg.org/eduVPN/eduvpn-common/internal/util"
	"codeberg.org/eduVPN/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

// AddSecure adds a secure internet server
// `ctx` is the context used for cancellation
// `disco` are the discovery servers
// `orgID` is the organiztaion ID
// `ot` specifies specifies the start time OAuth was already triggered
func (s *Servers) AddSecure(ctx context.Context, discom *discovery.Manager, orgID string, ot *int64) error {
	if s.config.HasSecureInternet() {
		return errors.New("a secure internet server already exists")
	}
	disco, release := discom.Discovery(false)
	dorg, dsrv, err := disco.SecureHomeArgs(orgID)
	if err != nil {
		release()
		return err
	}
	release()

	sd := api.ServerData{
		ID:         dorg.OrgID,
		Type:       server.TypeSecureInternet,
		BaseWK:     dsrv.BaseURL,
		BaseAuthWK: dsrv.BaseURL,
		ProcessAuth: func(ctx context.Context, url string) (string, error) {
			newd, release := discom.Discovery(true)
			defer release()
			// the only thing we can do is log warn
			// this is already done in the functions
			newd.Servers(ctx)       //nolint:errcheck
			newd.Organizations(ctx) //nolint:errcheck
			updorg, updsrv, err := newd.SecureHomeArgs(orgID)
			if err != nil {
				return "", err
			}
			ret := util.ReplaceWAYF(updsrv.AuthenticationURLTemplate, url, updorg.OrgID)
			return ret, nil
		},
	}

	auth := time.Time{}
	if ot != nil {
		auth = time.Unix(*ot, 0)
	}

	err = s.config.AddServer(orgID, server.TypeSecureInternet, v2.Server{
		CountryCode:       dsrv.CountryCode,
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
		rerr := s.config.RemoveServer(orgID, server.TypeSecureInternet)
		if rerr != nil {
			log.Logger.Warningf("could not remove secure internet server: '%s' after failing authorization: %v", orgID, rerr)
		}
		return err
	}
	return nil
}

// GetSecure gets a secure internet server
// `ctx` is the context used for cancellation
// `orgID` is the organization ID that identifies the server
// `disco` are the discovery servers
// `tok` are the tokens such that the server can be found without triggering auth
// `disableAuth` is set to true when authorization should not be triggered
func (s *Servers) GetSecure(ctx context.Context, orgID string, discom *discovery.Manager, tok *eduoauth.Token, disableAuth bool) (*Server, error) {
	srv, err := s.config.GetServer(orgID, server.TypeSecureInternet)
	if err != nil {
		return nil, err
	}

	disco, release := discom.Discovery(false)
	dorg, dhome, err := disco.SecureHomeArgs(orgID)
	if err != nil {
		release()
		return nil, err
	}

	dloc, err := disco.ServerByCountryCode(srv.CountryCode)
	if err != nil {
		release()
		return nil, err
	}
	release()

	sd := api.ServerData{
		ID:         dorg.OrgID,
		Type:       server.TypeSecureInternet,
		BaseWK:     dloc.BaseURL,
		BaseAuthWK: dhome.BaseURL,
		ProcessAuth: func(ctx context.Context, url string) (string, error) {
			newd, release := discom.Discovery(true)
			defer release()
			// the only thing we can do is log warn
			// this is already done in the functions
			newd.MarkServersExpired()
			newd.Servers(ctx) //nolint:errcheck
			newd.MarkOrganizationsExpired()
			newd.Organizations(ctx) //nolint:errcheck
			updorg, updsrv, err := newd.SecureHomeArgs(orgID)
			if err != nil {
				return "", err
			}
			ret := util.ReplaceWAYF(updsrv.AuthenticationURLTemplate, url, updorg.OrgID)
			return ret, nil
		},
		DisableAuthorize: disableAuth,
	}

	a, err := api.NewAPI(ctx, s.clientID, sd, s.cb, tok)
	if err != nil {
		return nil, err
	}

	sec := s.NewServer(orgID, server.TypeSecureInternet, a)
	return &sec, nil
}
