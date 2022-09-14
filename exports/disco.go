package main

/*
// for free
#include <stdlib.h>
#include "c/disco.h"
*/
import "C"

import (
	"unsafe"

	"github.com/jwijenbergh/eduvpn-common"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
)

func getCPtrDiscoOrganization(
	state *eduvpn.VPNState,
	organization *types.DiscoveryOrganization,
) *C.discoveryOrganization {
	returnedStruct := (*C.discoveryOrganization)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryOrganization{}))),
	)
	returnedStruct.display_name = C.CString(state.GetTranslated(organization.DisplayName))
	returnedStruct.org_id = C.CString(organization.OrgId)
	returnedStruct.secure_internet_home = C.CString(organization.SecureInternetHome)
	returnedStruct.keyword_list = C.CString(state.GetTranslated(organization.KeywordList))
	return returnedStruct
}

func getCPtrDiscoOrganizations(
	state *eduvpn.VPNState,
	organizations *types.DiscoveryOrganizations,
) (C.size_t, **C.discoveryOrganization) {
	totalOrganizations := C.size_t(len(organizations.List))
	var organizationsPtr **C.discoveryOrganization
	if totalOrganizations > 0 {
		organizationsPtr = (**C.discoveryOrganization)(
			C.malloc(totalOrganizations * C.size_t(unsafe.Sizeof(uintptr(0)))),
		)
		cOrganizations := (*[1<<30 - 1]*C.discoveryOrganization)(unsafe.Pointer(organizationsPtr))[:totalOrganizations:totalOrganizations]
		index := 0
		for _, organization := range organizations.List {
			cOrganization := getCPtrDiscoOrganization(state, &organization)
			cOrganizations[index] = cOrganization
			index += 1
		}
	}
	return totalOrganizations, organizationsPtr
}

func freeDiscoOrganization(cOrganization *C.discoveryOrganization) {
	C.free(unsafe.Pointer(cOrganization.display_name))
	C.free(unsafe.Pointer(cOrganization.org_id))
	C.free(unsafe.Pointer(cOrganization.secure_internet_home))
	C.free(unsafe.Pointer(cOrganization.keyword_list))
	C.free(unsafe.Pointer(cOrganization))
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

//export GetDiscoOrganizations
func GetDiscoOrganizations(name *C.char) *C.discoveryOrganizations {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	// TODO
	if stateErr != nil {
		panic(stateErr)
	}
	organizations, organizationsErr := state.GetDiscoOrganizations()
	// TODO
	if organizationsErr != nil {
		panic(organizationsErr)
	}

	returnedStruct := (*C.discoveryOrganizations)(
		C.malloc(C.size_t(unsafe.Sizeof(C.discoveryOrganizations{}))),
	)

	returnedStruct.version = C.ulonglong(organizations.Version)
	returnedStruct.total_organizations, returnedStruct.organizations = getCPtrDiscoOrganizations(
		state,
		organizations,
	)

	return returnedStruct
}
