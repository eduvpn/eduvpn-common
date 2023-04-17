package main

/*
// for free and size_t
#include <stdlib.h>
#include "error.h"
#include "server.h"
*/
import "C"

import (
	"unsafe"

	"github.com/eduvpn/eduvpn-common/client"
	"github.com/eduvpn/eduvpn-common/internal/server"
)

// Get the pointer to the C struct for the profile
// We allocate the struct, the profile ID and the display name
func getCPtrProfile(profile *server.Profile) *C.serverProfile {
	// Allocate the struct using malloc and the size of the struct
	cProfile := (*C.serverProfile)(C.malloc(C.size_t(unsafe.Sizeof(C.serverProfile{}))))
	cProfile.id = C.CString(profile.ID)
	cProfile.display_name = C.CString(profile.DisplayName)
	if profile.DefaultGateway {
		cProfile.default_gateway = C.int(1)
	} else {
		cProfile.default_gateway = C.int(0)
	}

	return cProfile
}

// Get the pointer to the C struct for the profiles
// We allocate the struct and the struct inside it for the profiles
func getCPtrProfiles(serverProfiles *server.ProfileInfo) *C.serverProfiles {
	goProfiles := serverProfiles.Info.ProfileList
	// Allocate the profles struct using malloc and the size of a pointer
	cProfiles := (*C.serverProfiles)(C.malloc(C.size_t(uintptr(0))))
	totalProfiles := C.size_t(len(goProfiles))
	// Defaults if we have no profiles
	cProfiles.current = C.int(0)
	cProfiles.profiles = nil
	cProfiles.total_profiles = totalProfiles
	// If we have profiles (which we should), we allocate the struct with malloc and the size of a pointer
	// We then fill the struct by converting it to a go slice and get a C pointer for each profile
	if totalProfiles > 0 {
		profilesPtr := (**C.serverProfile)(C.malloc(totalProfiles * C.size_t(unsafe.Sizeof(uintptr(0)))))
		profiles := unsafe.Slice(profilesPtr, totalProfiles)
		index := 0
		for _, profile := range goProfiles {
			profiles[index] = getCPtrProfile(&profile)
			index++
		}
		cProfiles.current = C.int(serverProfiles.CurrentProfileIndex())
		cProfiles.profiles = (**C.serverProfile)(profilesPtr)
	}
	return cProfiles
}

// Free the profiles by looping through them if there are any
// Also free the pointer itself
//
//export FreeProfiles
func FreeProfiles(profiles *C.serverProfiles) {
	// We should only free the profiles if we have them (which we should)
	if profiles.total_profiles > 0 {
		// Convert it to a go slice
		profilesSlice := unsafe.Slice(profiles.profiles, profiles.total_profiles)
		// Loop through the pointers and free th  allocated strings and the struct itself
		for i := C.size_t(0); i < profiles.total_profiles; i++ {
			C.free(unsafe.Pointer(profilesSlice[i].id))
			C.free(unsafe.Pointer(profilesSlice[i].display_name))
			C.free(unsafe.Pointer(profilesSlice[i]))
		}
		// Free the inner profiles struct
		C.free(unsafe.Pointer(profiles.profiles))
	}
	// Free the profiles struct itself
	C.free(unsafe.Pointer(profiles))
}

// Get a list of strings with a size as a c structure
// Returns the size in size_t and the list of strings as a double pointer char
func getCPtrListStrings(allStrings []string) (C.size_t, **C.char) {
	// Get the total strings in size_t
	totalStrings := C.size_t(len(allStrings))

	// If we have strings
	// Allocate memory for the strings array
	if totalStrings > 0 {
		stringsPtr := (**C.char)(C.malloc(totalStrings * C.size_t(unsafe.Sizeof(uintptr(0)))))
		// Go slice conversion
		cStrings := unsafe.Slice(stringsPtr, totalStrings)

		// Loop through and allocate the string for each contact
		for index, string := range allStrings {
			cStrings[index] = C.CString(string)
		}
		return totalStrings, (**C.char)(stringsPtr)
	}

	// No strings then the length is zero and the char array is nil
	return C.size_t(0), nil
}

// Function for freeing an array/list of strings
// It takes the strings as a pointer to a string and the total strings in size_t
func freeCListStrings(allStrings **C.char, totalStrings C.size_t) {
	// If we have strings we should free them
	// By converting to a Go slice, and freeing them ony by one
	// At last free the pointer itself
	if totalStrings > 0 {
		stringsSlice := unsafe.Slice(allStrings, totalStrings)
		for i := C.size_t(0); i < totalStrings; i++ {
			C.free(unsafe.Pointer(stringsSlice[i]))
		}
		C.free(unsafe.Pointer(allStrings))
	}
}

// Function for getting the server,
// It gets the main state as a pointer as we need to convert some string maps to localized strings
// It gets the base information for a server as well
func getCPtrServer(state *client.Client, base *client.ServerBase) *C.server {
	// Allocation using malloc and the size of the struct
	cServer := (*C.server)(C.malloc(C.size_t(unsafe.Sizeof(C.server{}))))
	// String allocation and translate the display name
	identifier := base.URL
	countryCode := ""
	// A secure internet server has multiple locations
	locations := []string{}
	if base.Type == "secure_internet" {
		identifier = state.Servers.SecureInternetHomeServer.HomeOrganizationID
		countryCode = state.Servers.SecureInternetHomeServer.CurrentLocation
		locations = state.Discovery.SecureLocationList()
	}

	cServer.identifier = C.CString(identifier)
	cServer.display_name = C.CString(state.GetTranslated(base.DisplayName))
	cServer.country_code = C.CString(countryCode)
	cServer.server_type = C.CString(base.Type)
	// Call the helper to get the list of support contacts
	cServer.total_support_contact, cServer.support_contact = getCPtrListStrings(
		base.SupportContact,
	)
	locationsStruct := (*C.serverLocations)(C.malloc(C.size_t(unsafe.Sizeof(C.servers{}))))
	locationsStruct.total_locations, locationsStruct.locations = getCPtrListStrings(locations)
	cServer.locations = locationsStruct

	profiles := base.ValidProfiles(state.SupportsWireguard)
	cServer.profiles = getCPtrProfiles(&profiles)
	// No endtime is given if we get servers when it has been partially initialised
	if base.EndTime.IsZero() {
		cServer.expire_time = C.ulonglong(0)
	} else {
		// The expire time should be stored as an unsigned long long in unix time
		cServer.expire_time = C.ulonglong(base.EndTime.Unix())
	}
	return cServer
}

// Function for freeing a single server
// Gets the pointer to C struct
//
//export FreeServer
func FreeServer(info *C.server) {
	// Free strings
	C.free(unsafe.Pointer(info.identifier))
	C.free(unsafe.Pointer(info.display_name))
	C.free(unsafe.Pointer(info.country_code))
	C.free(unsafe.Pointer(info.server_type))

	// Free arrays
	freeCListStrings(info.support_contact, info.total_support_contact)
	FreeSecureLocations(info.locations)
	FreeProfiles(info.profiles)

	// Free the struct itself
	C.free(unsafe.Pointer(info))
}

// Get the C ptr to the servers, returns the length in size_t and the double pointer to the struct
func getCPtrServers(
	state *client.Client,
	serverMap map[string]*server.InstituteAccessServer,
) (C.size_t, **C.server) {
	totalServers := C.size_t(len(serverMap))
	// If we have servers, which is not always the case
	if totalServers > 0 {
		serversPtr := (**C.server)(C.malloc(totalServers * C.size_t(unsafe.Sizeof(uintptr(0)))))
		servers := unsafe.Slice(serversPtr, totalServers)
		index := 0
		for _, currentServer := range serverMap {
			cServer := getCPtrServer(state, &currentServer.Basic)
			servers[index] = cServer
			index++
		}
		return totalServers, serversPtr
	}
	return C.size_t(0), nil
}

// This function takes the servers as a C struct pointer as input
// It frees all allocated memory for the server
//
//export FreeServers
func FreeServers(cServers *C.servers) {
	// Free the custom servers if there are any
	if cServers.total_custom > 0 {
		customServers := unsafe.Slice(cServers.custom_servers, cServers.total_custom)
		for i := C.size_t(0); i < cServers.total_custom; i++ {
			FreeServer(customServers[i])
		}
		C.free(unsafe.Pointer(cServers.custom_servers))
	}
	// Free the institute access servers if there are any
	if cServers.total_institute > 0 {
		instituteServers := unsafe.Slice(cServers.institute_servers, cServers.total_institute)

		for i := C.size_t(0); i < cServers.total_institute; i++ {
			FreeServer(instituteServers[i])
		}
		C.free(unsafe.Pointer(cServers.institute_servers))
	}
	// Free the secure internet server if there is one
	if cServers.secure_internet_server != nil {
		FreeServer(cServers.secure_internet_server)
	}
	// Free the structure itself
	C.free(unsafe.Pointer(cServers))
}

// Return the servers as a C struct pointer
// It takes the state as a pointer as we need to translate some strings
// It also takes the servers as a pointer that belongs to the main state or gathered from the callback
func getSavedServersWithOptions(state *client.Client, servers *server.Servers) *C.servers {
	// Allocate the struct that we will return
	// With the size of the c struct
	returnedStruct := (*C.servers)(C.malloc(C.size_t(unsafe.Sizeof(C.servers{}))))

	// Get the different categories of servers
	totalCustom, customPtr := getCPtrServers(state, servers.CustomServers.Map)
	totalInstitute, institutePtr := getCPtrServers(state, servers.InstituteServers.Map)
	var secureServerPtr *C.server
	secureInternetBase, secureInternetBaseErr := servers.SecureInternetHomeServer.Base()
	if secureInternetBaseErr == nil && secureInternetBase != nil {
		// FIXME: log error?
		secureServerPtr = getCPtrServer(state, secureInternetBase)
		// Give a new identifier
		C.free(unsafe.Pointer(secureServerPtr.identifier))
		secureServerPtr.identifier = C.CString(servers.SecureInternetHomeServer.HomeOrganizationID)
		secureServerPtr.country_code = C.CString(servers.SecureInternetHomeServer.CurrentLocation)
	}

	// Fill the struct and return
	returnedStruct.custom_servers = customPtr
	returnedStruct.total_custom = totalCustom
	returnedStruct.institute_servers = institutePtr
	returnedStruct.total_institute = totalInstitute
	returnedStruct.secure_internet_server = secureServerPtr
	return returnedStruct
}

// This function takes the name as input which is the name of the client
// It gets the state by name and then returns the saved servers as a c struct belonging to it
//
//export GetSavedServers
func GetSavedServers(name *C.char) (*C.servers, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, getError(stateErr)
	}
	servers := getSavedServersWithOptions(state, &state.Servers)
	return servers, nil
}

// This function takes the name as input which is the name of the client
// It gets the state by name and then returns the current server as a c struct belonging to it
//
//export GetCurrentServer
func GetCurrentServer(name *C.char) (*C.server, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, getError(stateErr)
	}
	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		return nil, getError(serverErr)
	}
	base, baseErr := server.Base()
	if baseErr != nil {
		return nil, getError(baseErr)
	}
	cServer := getCPtrServer(state, base)
	return cServer, nil
}

// This function takes the state as input which is the main state
// It also takes the data as an interface and if it has the servers type gets the data as a c struct otherwise nil
func getTransitionDataServers(state *client.Client, data interface{}) *C.servers {
	if converted, ok := data.(server.Servers); ok {
		return getSavedServersWithOptions(state, &converted)
	}
	return nil
}

//export FreeSecureLocations
func FreeSecureLocations(locations *C.serverLocations) {
	freeCListStrings(locations.locations, locations.total_locations)
	C.free(unsafe.Pointer(locations))
}

func getTransitionSecureLocations(data interface{}) *C.serverLocations {
	if locations, ok := data.([]string); ok {
		returnedStruct := (*C.serverLocations)(C.malloc(C.size_t(unsafe.Sizeof(C.servers{}))))
		returnedStruct.total_locations, returnedStruct.locations = getCPtrListStrings(locations)
		return returnedStruct
	}
	return nil
}

func getTransitionProfiles(data interface{}) *C.serverProfiles {
	if profiles, ok := data.(*server.ProfileInfo); ok {
		return getCPtrProfiles(profiles)
	}
	return nil
}

func getTransitionServer(state *client.Client, data interface{}) *C.server {
	if server, ok := data.(server.Server); ok {
		base, baseErr := server.Base()
		if baseErr != nil {
			// TODO: LOG
			return nil
		}
		return getCPtrServer(state, base)
	}
	return nil
}
