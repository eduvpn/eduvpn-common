package server

import (
	"fmt"

	"github.com/eduvpn/eduvpn-common/types"
)

type Servers struct {
	// A custom server is just an institute access server under the hood
	CustomServers            InstituteAccessServers   `json:"custom_servers"`
	InstituteServers         InstituteAccessServers   `json:"institute_servers"`
	SecureInternetHomeServer SecureInternetHomeServer `json:"secure_internet_home"`
	IsType                   Type                     `json:"is_secure_internet"`
}

func (servers *Servers) AddSecureInternet(
	secureOrg *types.DiscoveryOrganization,
	secureServer *types.DiscoveryServer,
) (Server, error) {
	errorMessage := "failed adding secure internet server"
	// If we have specified an organization ID
	// We also need to get an authorization template
	initErr := servers.SecureInternetHomeServer.init(secureOrg, secureServer)

	if initErr != nil {
		return nil, types.NewWrappedError(errorMessage, initErr)
	}

	servers.IsType = SecureInternetServerType
	return &servers.SecureInternetHomeServer, nil
}

func (servers *Servers) GetCurrentServer() (Server, error) {
	errorMessage := "failed getting current server"
	if servers.IsType == SecureInternetServerType {
		if !servers.HasSecureLocation() {
			return nil, types.NewWrappedError(
				errorMessage,
				&CurrentNotFoundError{},
			)
		}
		return &servers.SecureInternetHomeServer, nil
	}

	serversStruct := &servers.InstituteServers

	if servers.IsType == CustomServerType {
		serversStruct = &servers.CustomServers
	}
	currentServerURL := serversStruct.CurrentURL
	bases := serversStruct.Map
	if bases == nil {
		return nil, types.NewWrappedError(
			errorMessage,
			&CurrentNoMapError{},
		)
	}
	server, exists := bases[currentServerURL]

	if !exists || server == nil {
		return nil, types.NewWrappedError(
			errorMessage,
			&CurrentNotFoundError{},
		)
	}
	return server, nil
}

func (servers *Servers) addInstituteAndCustom(
	discoServer *types.DiscoveryServer,
	isCustom bool,
) (Server, error) {
	url := discoServer.BaseURL
	errorMessage := fmt.Sprintf("failed adding institute access server: %s", url)
	toAddServers := &servers.InstituteServers
	serverType := InstituteAccessServerType

	if isCustom {
		toAddServers = &servers.CustomServers
		serverType = CustomServerType
	}

	if toAddServers.Map == nil {
		toAddServers.Map = make(map[string]*InstituteAccessServer)
	}

	server, exists := toAddServers.Map[url]

	// initialize the server if it doesn't exist yet
	if !exists {
		server = &InstituteAccessServer{}
	}

	instituteInitErr := server.init(
		url,
		discoServer.DisplayName,
		discoServer.Type,
		discoServer.SupportContact,
	)
	if instituteInitErr != nil {
		return nil, types.NewWrappedError(errorMessage, instituteInitErr)
	}
	toAddServers.Map[url] = server
	servers.IsType = serverType
	return server, nil
}

func (servers *Servers) AddInstituteAccessServer(
	instituteServer *types.DiscoveryServer,
) (Server, error) {
	return servers.addInstituteAndCustom(instituteServer, false)
}

func (servers *Servers) AddCustomServer(
	customServer *types.DiscoveryServer,
) (Server, error) {
	return servers.addInstituteAndCustom(customServer, true)
}

func (servers *Servers) GetSecureLocation() string {
	return servers.SecureInternetHomeServer.CurrentLocation
}

func (servers *Servers) SetSecureLocation(
	chosenLocationServer *types.DiscoveryServer,
) error {
	errorMessage := "failed to set secure location"
	// Make sure to add the current location
	_, addLocationErr := servers.SecureInternetHomeServer.addLocation(chosenLocationServer)

	if addLocationErr != nil {
		return types.NewWrappedError(errorMessage, addLocationErr)
	}

	servers.SecureInternetHomeServer.CurrentLocation = chosenLocationServer.CountryCode
	return nil
}
