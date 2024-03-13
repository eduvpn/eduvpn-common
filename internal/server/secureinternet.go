package server

import (
	"context"
	"errors"

	"github.com/eduvpn/eduvpn-common/internal/api"
	"github.com/eduvpn/eduvpn-common/internal/config/v2"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/internal/util"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

// AddSecure adds a secure internet server
// `ctx` is the context used for cancellation
// `disco` are the discovery servers
// `orgID` is the organiztaion ID
// `na` specifies whether or not authorization should be triggered when adding
func (s *Servers) AddSecure(ctx context.Context, disco *discovery.Discovery, orgID string, na bool) error {
	if s.config.HasSecureInternet() {
		return errors.New("a secure internet server already exists")
	}
	dorg, dsrv, err := disco.SecureHomeArgs(orgID)
	if err != nil {
		// We mark the organizations as expired because we got an error
		// Note that in the docs it states that it only should happen when the Org ID doesn't exist
		// However, this is nice as well because it also catches the error where the SecureInternetHome server is not found
		disco.MarkOrganizationsExpired()
		return err
	}

	sd := api.ServerData{
		ID:         orgID,
		Type:       server.TypeSecureInternet,
		BaseWK:     dsrv.BaseURL,
		BaseAuthWK: dsrv.BaseURL,
		ProcessAuth: func(url string) string {
			return util.ReplaceWAYF(dsrv.AuthenticationURLTemplate, url, dorg.OrgID)
		},
	}

	err = s.config.AddServer(orgID, server.TypeSecureInternet, v2.Server{CountryCode: dsrv.CountryCode})
	if err != nil {
		return err
	}

	// no authorization should be triggered, return
	if na {
		return nil
	}

	// Authorize by creating the API object
	_, err = api.NewAPI(ctx, s.clientID, sd, s.cb, nil)
	if err != nil {
		// authorization has failed, remove the server again
		s.config.RemoveServer(orgID, server.TypeSecureInternet)
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
func (s *Servers) GetSecure(ctx context.Context, orgID string, disco *discovery.Discovery, tok *eduoauth.Token, disableAuth bool) (*Server, error) {
	srv, err := s.config.GetServer(orgID, server.TypeSecureInternet)
	if err != nil {
		return nil, err
	}

	dorg, dhome, err := disco.SecureHomeArgs(orgID)
	if err != nil {
		return nil, err
	}

	dloc, err := disco.ServerByCountryCode(srv.CountryCode)
	if err != nil {
		return nil, err
	}

	sd := api.ServerData{
		ID:         dorg.OrgID,
		Type:       server.TypeSecureInternet,
		BaseWK:     dloc.BaseURL,
		BaseAuthWK: dhome.BaseURL,
		ProcessAuth: func(url string) string {
			return util.ReplaceWAYF(dhome.AuthenticationURLTemplate, url, dorg.OrgID)
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
