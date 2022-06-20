package main

/*
#include <stdlib.h>

typedef void (*PythonCB)(const char* oldstate, const char* newstate, const char* data);

__attribute__((weak))
void call_callback(PythonCB callback, const char* oldstate, const char* newstate, const char* data)
{
    callback(oldstate, newstate, data);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/jwijenbergh/eduvpn-common"
)

var P_StateCallback C.PythonCB

var VPNStates map[string]*eduvpn.VPNState

func StateCallback(old_state string, new_state string, data string) {
	if P_StateCallback == nil {
		return
	}
	oldState_c := C.CString(old_state)
	newState_c := C.CString(new_state)
	data_c := C.CString(data)
	C.call_callback(P_StateCallback, oldState_c, newState_c, data_c)
	C.free(unsafe.Pointer(oldState_c))
	C.free(unsafe.Pointer(newState_c))
	C.free(unsafe.Pointer(data_c))
}

func GetVPNState(name string) (*eduvpn.VPNState, error) {
	state, exists := VPNStates[name]

	if !exists || state == nil {
		return nil, errors.New(fmt.Sprintf("State with name %s not found", name))
	}

	return state, nil
}

//export Register
func Register(name *C.char, config_directory *C.char, stateCallback C.PythonCB, debug C.int) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		state = &eduvpn.VPNState{}
	}
	if VPNStates == nil {
		VPNStates = make(map[string]*eduvpn.VPNState)
	}
	VPNStates[nameStr] = state
	P_StateCallback = stateCallback
	registerErr := state.Register(nameStr, C.GoString(config_directory), StateCallback, debug != 0)

	if registerErr != nil {
		delete(VPNStates, nameStr)
	}
	return C.CString(ErrorToString(registerErr))
}

//export Deregister
func Deregister(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	state.Deregister()
	return nil
}

func ErrorToString(error error) string {
	if error == nil {
		return ""
	}

	return error.Error()
}

//export CancelOAuth
func CancelOAuth(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	cancelErr := state.CancelOAuth()
	cancelErrString := ErrorToString(cancelErr)
	return C.CString(cancelErrString)
}

//export GetConnectConfig
func GetConnectConfig(name *C.char, url *C.char, isSecureInternet C.int, forceTCP C.int) (*C.char, *C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, nil, C.CString(ErrorToString(stateErr))
	}
	var config string
	var configType string
	var configErr error
	forceTCPBool := forceTCP == 1
	if isSecureInternet == 1 {
		config, configType, configErr = state.GetConfigSecureInternet(C.GoString(url), forceTCPBool)
	} else {
		config, configType, configErr = state.GetConfigInstituteAccess(C.GoString(url), forceTCPBool)
	}
	return C.CString(config), C.CString(configType), C.CString(ErrorToString(configErr))
}

//export GetOrganizationsList
func GetOrganizationsList(name *C.char) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	organizations, organizationsErr := state.GetDiscoOrganizations()
	return C.CString(organizations), C.CString(ErrorToString(organizationsErr))
}

//export GetServersList
func GetServersList(name *C.char) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	servers, serversErr := state.GetDiscoServers()
	return C.CString(servers), C.CString(ErrorToString(serversErr))
}

//export SetProfileID
func SetProfileID(name *C.char, data *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	profileErr := state.SetProfileID(C.GoString(data))
	return C.CString(ErrorToString(profileErr))
}

//export SetDisconnected
func SetDisconnected(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setDisconnectedErr := state.SetDisconnected()
	return C.CString(ErrorToString(setDisconnectedErr))
}

//export SetConnected
func SetConnected(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setConnectedErr := state.SetConnected()
	return C.CString(ErrorToString(setConnectedErr))
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
