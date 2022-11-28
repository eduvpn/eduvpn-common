package main

/*
// for free and size_t
#include <stdlib.h>
#include "error.h"

typedef struct discoveryServer {
  const char* authentication_url_template;
  const char* base_url;
  const char* country_code;
  const char* display_name;
  const char* keyword_list;
  const char** public_key_list;
  size_t total_public_keys;
  const char* server_type;
  const char** support_contact;
  size_t total_support_contact;
} discoveryServer;

typedef struct discoveryServers {
  unsigned long long int version;
  discoveryServer** servers;
  size_t total_servers;
} discoveryServers;

typedef struct discoveryOrganization {
  const char* display_name;
  const char* org_id;
  const char* secure_internet_home;
  const char* keyword_list;
} discoveryOrganization;

typedef struct discoveryOrganizations {
  unsigned long long int version;
  discoveryOrganization** organizations;
  size_t total_organizations;
} discoveryOrganizations;
*/
import "C"

import (
	"unsafe"

	"github.com/eduvpn/eduvpn-common/client"
	"github.com/eduvpn/eduvpn-common/types"
)

func getCPtrDiscoOrganization(
	state *client.Client,
	organization *types.DiscoveryOrganization,
) *C.discoveryOrganization {
	returnedStruct := (*C.discoveryOrganization)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryOrganization{}))),
	)
	returnedStruct.display_name = C.CString(state.GetTranslated(organization.DisplayName))
	returnedStruct.org_id = C.CString(organization.OrgID)
	returnedStruct.secure_internet_home = C.CString(organization.SecureInternetHome)
	returnedStruct.keyword_list = C.CString(state.GetTranslated(organization.KeywordList))
	return returnedStruct
}

func getCPtrDiscoOrganizations(
	state *client.Client,
	organizations *types.DiscoveryOrganizations,
) (C.size_t, **C.discoveryOrganization) {
	totalOrganizations := C.size_t(len(organizations.List))
	if totalOrganizations > 0 {
		organizationsPtr := (**C.discoveryOrganization)(
			C.malloc(totalOrganizations * C.size_t(unsafe.Sizeof(uintptr(0)))),
		)
		cOrganizations := (*[1<<30 - 1]*C.discoveryOrganization)(unsafe.Pointer(organizationsPtr))[:totalOrganizations:totalOrganizations]
		index := 0
		for _, organization := range organizations.List {
			cOrganization := getCPtrDiscoOrganization(state, &organization)
			cOrganizations[index] = cOrganization
			index++
		}
		return totalOrganizations, organizationsPtr
	}
	return 0, nil
}

func getCPtrDiscoServer(
	state *client.Client,
	server *types.DiscoveryServer,
) *C.discoveryServer {
	returnedStruct := (*C.discoveryServer)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryServer{}))),
	)
	returnedStruct.authentication_url_template = C.CString(server.AuthenticationURLTemplate)
	returnedStruct.base_url = C.CString(server.BaseURL)
	returnedStruct.country_code = C.CString(server.CountryCode)
	returnedStruct.display_name = C.CString(state.GetTranslated(server.DisplayName))
	returnedStruct.keyword_list = C.CString(state.GetTranslated(server.KeywordList))
	returnedStruct.total_public_keys, returnedStruct.public_key_list = getCPtrListStrings(
		server.PublicKeyList,
	)
	returnedStruct.server_type = C.CString(server.Type)
	returnedStruct.total_support_contact, returnedStruct.support_contact = getCPtrListStrings(
		server.SupportContact,
	)
	return returnedStruct
}

func getCPtrDiscoServers(
	state *client.Client,
	servers *types.DiscoveryServers,
) (C.size_t, **C.discoveryServer) {
	totalServers := C.size_t(len(servers.List))
	if totalServers > 0 {
		serversPtr := (**C.discoveryServer)(
			C.malloc(totalServers * C.size_t(unsafe.Sizeof(uintptr(0)))),
		)
		cServers := (*[1<<30 - 1]*C.discoveryServer)(unsafe.Pointer(serversPtr))
		index := 0
		for _, server := range servers.List {
			cServer := getCPtrDiscoServer(state, &server)
			cServers[index] = cServer
			index++
		}
		return totalServers, serversPtr
	}
	return 0, nil
}

func freeDiscoOrganization(cOrganization *C.discoveryOrganization) {
	C.free(unsafe.Pointer(cOrganization.display_name))
	C.free(unsafe.Pointer(cOrganization.org_id))
	C.free(unsafe.Pointer(cOrganization.secure_internet_home))
	C.free(unsafe.Pointer(cOrganization.keyword_list))
	C.free(unsafe.Pointer(cOrganization))
}

func freeDiscoServer(cServer *C.discoveryServer) {
	C.free(unsafe.Pointer(cServer.authentication_url_template))
	C.free(unsafe.Pointer(cServer.base_url))
	C.free(unsafe.Pointer(cServer.country_code))
	C.free(unsafe.Pointer(cServer.display_name))
	C.free(unsafe.Pointer(cServer.keyword_list))
	freeCListStrings(cServer.public_key_list, cServer.total_public_keys)
	C.free(unsafe.Pointer(cServer.server_type))
	freeCListStrings(cServer.support_contact, cServer.total_support_contact)
	C.free(unsafe.Pointer(cServer))
}

//export FreeDiscoServers
func FreeDiscoServers(cServers *C.discoveryServers) {
	if cServers.total_servers > 0 {
		servers := (*[1<<30 - 1]*C.discoveryServer)(unsafe.Pointer(cServers.servers))[:cServers.total_servers:cServers.total_servers]
		for i := C.size_t(0); i < cServers.total_servers; i++ {
			freeDiscoServer(servers[i])
		}
		C.free(unsafe.Pointer(cServers.servers))
	}
	C.free(unsafe.Pointer(cServers))
}

//export FreeDiscoOrganizations
func FreeDiscoOrganizations(cOrganizations *C.discoveryOrganizations) {
	if cOrganizations.total_organizations > 0 {
		organizations := (*[1<<30 - 1]*C.discoveryOrganization)(unsafe.Pointer(cOrganizations.organizations))[:cOrganizations.total_organizations:cOrganizations.total_organizations]
		for i := C.size_t(0); i < cOrganizations.total_organizations; i++ {
			freeDiscoOrganization(organizations[i])
		}
		C.free(unsafe.Pointer(cOrganizations.organizations))
	}
	C.free(unsafe.Pointer(cOrganizations))
}

//export GetDiscoServers
func GetDiscoServers(name *C.char) (*C.discoveryServers, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, getError(stateErr)
	}
	servers, serversErr := state.DiscoServers()
	if serversErr != nil {
		return nil, getError(serversErr)
	}

	returnedStruct := (*C.discoveryServers)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryServers{}))),
	)
	returnedStruct.version = C.ulonglong(servers.Version)
	returnedStruct.total_servers, returnedStruct.servers = getCPtrDiscoServers(
		state,
		servers,
	)
	return returnedStruct, nil
}

//export GetDiscoOrganizations
func GetDiscoOrganizations(name *C.char) (*C.discoveryOrganizations, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, getError(stateErr)
	}
	organizations, organizationsErr := state.DiscoOrganizations()
	if organizationsErr != nil {
		return nil, getError(organizationsErr)
	}

	returnedStruct := (*C.discoveryOrganizations)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryOrganizations{}))),
	)

	returnedStruct.version = C.ulonglong(organizations.Version)
	returnedStruct.total_organizations, returnedStruct.organizations = getCPtrDiscoOrganizations(
		state,
		organizations,
	)

	return returnedStruct, nil
}
