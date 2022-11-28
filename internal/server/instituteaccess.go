package server

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/types"
)

// An instute access server.
type InstituteAccessServer struct {
	// An instute access server has its own OAuth
	Auth oauth.OAuth `json:"oauth"`

	// Embed the server base
	Basic Base `json:"base"`
}

type InstituteAccessServers struct {
	Map        map[string]*InstituteAccessServer `json:"map"`
	CurrentURL string                            `json:"current_url"`
}

func (servers *Servers) SetInstituteAccess(server Server) error {
	errorMessage := "failed setting institute access server"
	base, baseErr := server.Base()
	if baseErr != nil {
		return types.NewWrappedError(errorMessage, baseErr)
	}

	if base.Type != "institute_access" {
		return types.NewWrappedError(errorMessage, errors.New("not an institute access server"))
	}

	if _, ok := servers.InstituteServers.Map[base.URL]; ok {
		servers.InstituteServers.CurrentURL = base.URL
		servers.IsType = InstituteAccessServerType
	} else {
		return types.NewWrappedError(errorMessage, errors.New("no such institute access server"))
	}
	return nil
}

func (servers *Servers) GetInstituteAccess(url string) (*InstituteAccessServer, error) {
	if server, ok := servers.InstituteServers.Map[url]; ok {
		return server, nil
	}
	return nil, types.NewWrappedError("failed to get institute access server", fmt.Errorf("no institute access server with URL: %s", url))
}

func (servers *Servers) RemoveInstituteAccess(url string) {
	servers.InstituteServers.Remove(url)
}

func (servers *InstituteAccessServers) Remove(url string) {
	// Reset the current url
	if servers.CurrentURL == url {
		servers.CurrentURL = ""
	}

	// Delete the url from the map
	delete(servers.Map, url)
}

func (institute *InstituteAccessServer) TemplateAuth() func(string) string {
	return func(authURL string) string {
		return authURL
	}
}

func (institute *InstituteAccessServer) Base() (*Base, error) {
	return &institute.Basic, nil
}

func (institute *InstituteAccessServer) OAuth() *oauth.OAuth {
	return &institute.Auth
}

func (institute *InstituteAccessServer) init(
	url string,
	displayName map[string]string,
	serverType string,
	supportContact []string,
) error {
	errorMessage := fmt.Sprintf("failed initializing server %s", url)
	institute.Basic.URL = url
	institute.Basic.DisplayName = displayName
	institute.Basic.SupportContact = supportContact
	institute.Basic.Type = serverType
	endpointsErr := institute.Basic.InitializeEndpoints()
	if endpointsErr != nil {
		return types.NewWrappedError(errorMessage, endpointsErr)
	}
	API := institute.Basic.Endpoints.API.V3
	institute.Auth.Init(url, API.Authorization, API.Token)
	return nil
}
