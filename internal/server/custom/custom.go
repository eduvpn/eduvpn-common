package custom

import (
	"context"
	"net/url"

	"github.com/eduvpn/eduvpn-common/internal/server/api"
	"github.com/eduvpn/eduvpn-common/internal/server/base"
	"github.com/eduvpn/eduvpn-common/internal/server/institute"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/go-errors/errors"
)

type (
	Server  = institute.Server
	Servers = institute.Servers
)

func New(ctx context.Context, clientID string, u string) (*Server, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return nil, errors.WrapPrefix(err, "failed to parse custom server URL", 0)
	}
	b := base.Base{
		URL:         u,
		DisplayName: map[string]string{"en": pu.Hostname()},
		Type:        server.TypeCustom,
	}
	if err := api.Endpoints(ctx, &b); err != nil {
		return nil, err
	}
	API := b.Endpoints.API.V3

	s := &Server{Basic: b}
	// we set ISS to empty here as we do not want to have ISS enabled for custom servers
	// Otherwise we would have to normalise the URL which the user has entered which is error prone
	s.Auth.Init(clientID, "", API.Authorization, API.Token)
	return s, nil
}
