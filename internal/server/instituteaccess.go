package server

import (
	"errors"
	"fmt"

	"github.com/eduvpn/eduvpn-common/internal/oauth"
	"github.com/eduvpn/eduvpn-common/types"
)

// An instute access server
type InstituteAccessServer struct {
	// An instute access server has its own OAuth
	OAuth oauth.OAuth `json:"oauth"`

	// Embed the server base
	Base ServerBase `json:"base"`
}

type InstituteAccessServers struct {
	Map        map[string]*InstituteAccessServer `json:"map"`
	CurrentURL string                            `json:"current_url"`
}

func (servers *Servers) SetInstituteAccess(server Server) error {
	errorMessage := "failed setting institute access server"
	base, baseErr := server.GetBase()
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

// For an institute, we can simply get the OAuth
func (institute *InstituteAccessServer) GetOAuth() *oauth.OAuth {
	return &institute.OAuth
}

func (institute *InstituteAccessServer) GetTemplateAuth() func(string) string {
	return func(authURL string) string {
		return authURL
	}
}

func (institute *InstituteAccessServer) GetBase() (*ServerBase, error) {
	return &institute.Base, nil
}

func (institute *InstituteAccessServer) init(
	url string,
	displayName map[string]string,
	serverType string,
	supportContact []string,
) error {
	errorMessage := fmt.Sprintf("failed initializing server %s", url)
	institute.Base.URL = url
	institute.Base.DisplayName = displayName
	institute.Base.SupportContact = supportContact
	institute.Base.Type = serverType
	endpointsErr := institute.Base.InitializeEndpoints()
	if endpointsErr != nil {
		return types.NewWrappedError(errorMessage, endpointsErr)
	}
	API := institute.Base.Endpoints.API.V3
	institute.OAuth.Init(url, API.Authorization, API.Token)
	return nil
}
