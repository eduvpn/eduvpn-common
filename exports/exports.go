package main

/*
#include <stdlib.h>
#include "error.h"

typedef int (*PythonCB)(const char* name, int oldstate, int newstate, void* data);

static int call_callback(PythonCB callback, const char *name, int oldstate, int newstate, void* data)
{
    return callback(name, oldstate, newstate, data);
}
*/
import "C"

import (
	"github.com/eduvpn/eduvpn-common/internal/log"
	"github.com/go-errors/errors"
	"unsafe"

	"github.com/eduvpn/eduvpn-common/client"
)

var PStateCallbacks map[string]C.PythonCB

var VPNStates map[string]*client.Client

func GetStateData(
	state *client.Client,
	stateID client.FSMStateID,
	data interface{},
) unsafe.Pointer {
	switch stateID {
	case client.StateNoServer:
		return (unsafe.Pointer)(getTransitionDataServers(state, data))
	case client.StateOAuthStarted:
		if converted, ok := data.(string); ok {
			return (unsafe.Pointer)(C.CString(converted))
		}
	case client.StateAskLocation:
		return (unsafe.Pointer)(getTransitionSecureLocations(data))
	case client.StateAskProfile:
		return (unsafe.Pointer)(getTransitionProfiles(data))
	case client.StateDisconnected:
		return (unsafe.Pointer)(getTransitionServer(state, data))
	case client.StateDisconnecting:
		return (unsafe.Pointer)(getTransitionServer(state, data))
	case client.StateConnecting:
		return (unsafe.Pointer)(getTransitionServer(state, data))
	case client.StateConnected:
		return (unsafe.Pointer)(getTransitionServer(state, data))
	default:
		return nil
	}
	return nil
}

func StateCallback(
	state *client.Client,
	name string,
	oldState client.FSMStateID,
	newState client.FSMStateID,
	data interface{},
) bool {
	PStateCallback, exists := PStateCallbacks[name]
	if !exists || PStateCallback == nil {
		return false
	}
	nameC := C.CString(name)
	oldStateC := C.int(oldState)
	newStateC := C.int(newState)
	dataC := GetStateData(state, newState, data)
	handled := C.call_callback(PStateCallback, nameC, oldStateC, newStateC, dataC)
	C.free(unsafe.Pointer(nameC))
	// data_c gets freed by the wrapper
	return handled == C.int(1)
}

func GetVPNState(name string) (*client.Client, error) {
	state, exists := VPNStates[name]

	if !exists || state == nil {
		return nil, errors.Errorf("state with name %s not found", name)
	}

	return state, nil
}

//export Register
func Register(
	name *C.char,
	configDirectory *C.char,
	language *C.char,
	stateCallback C.PythonCB,
	debug C.int,
) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		state = &client.Client{}
	}
	if VPNStates == nil {
		VPNStates = make(map[string]*client.Client)
	}
	if PStateCallbacks == nil {
		PStateCallbacks = make(map[string]C.PythonCB)
	}
	VPNStates[nameStr] = state
	PStateCallbacks[nameStr] = stateCallback
	registerErr := state.Register(
		nameStr,
		C.GoString(configDirectory),
		C.GoString(language),
		func(old client.FSMStateID, new client.FSMStateID, data interface{}) bool {
			return StateCallback(state, nameStr, old, new, data)
		},
		debug != 0,
	)

	if registerErr != nil {
		delete(VPNStates, nameStr)
	}
	return getError(registerErr)
}

//export Deregister
func Deregister(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	state.Deregister()
	return nil
}

func getError(err error) *C.error {
	if err == nil {
		return nil
	}
	errorStruct := (*C.error)(
		C.malloc(C.size_t(unsafe.Sizeof(C.error{}))),
	)
	if err1, ok := err.(*errors.Error); ok {
		errorStruct.traceback = C.CString(err1.ErrorStack())
		if err1.Err == nil {
			errorStruct.cause = C.CString(err1.Error())
		} else {
			errorStruct.cause = C.CString(err1.Err.Error())
		}
	} else {
		errorStruct.traceback = C.CString("N/A")
		errorStruct.cause = C.CString(err.Error())
	}
	errorStruct.level = C.errorLevel(log.GetErrorLevel(err))
	return errorStruct
}

//export FreeError
func FreeError(err *C.error) {
	C.free(unsafe.Pointer(err.traceback))
	C.free(unsafe.Pointer(err.cause))
	C.free(unsafe.Pointer(err))
}

//export CancelOAuth
func CancelOAuth(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	cancelErr := state.CancelOAuth()
	return getError(cancelErr)
}

//export RemoveSecureInternet
func RemoveSecureInternet(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	removeErr := state.RemoveSecureInternet()
	return getError(removeErr)
}

//export AddInstituteAccess
func AddInstituteAccess(name *C.char, url *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	// FIXME: Return server result
	_, addErr := state.AddInstituteServer(C.GoString(url))
	return getError(addErr)
}

//export AddSecureInternetHomeServer
func AddSecureInternetHomeServer(name *C.char, orgID *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	// FIXME: Return server result
	_, addErr := state.AddSecureInternetHomeServer(C.GoString(orgID))
	return getError(addErr)
}

//export AddCustomServer
func AddCustomServer(name *C.char, url *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	// FIXME: Return server result
	_, addErr := state.AddCustomServer(C.GoString(url))
	return getError(addErr)
}

//export RemoveInstituteAccess
func RemoveInstituteAccess(name *C.char, url *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	removeErr := state.RemoveInstituteAccess(C.GoString(url))
	return getError(removeErr)
}

//export RemoveCustomServer
func RemoveCustomServer(name *C.char, url *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	removeErr := state.RemoveCustomServer(C.GoString(url))
	return getError(removeErr)
}

//export GetConfigSecureInternet
func GetConfigSecureInternet(
	name *C.char,
	orgID *C.char,
	preferTCP C.int,
) (*C.char, *C.char, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, nil, getError(stateErr)
	}
	preferTCPBool := preferTCP == 1
	config, configType, configErr := state.GetConfigSecureInternet(C.GoString(orgID), preferTCPBool)
	return C.CString(config), C.CString(configType), getError(configErr)
}

//export GetConfigInstituteAccess
func GetConfigInstituteAccess(
	name *C.char,
	url *C.char,
	preferTCP C.int,
) (*C.char, *C.char, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, nil, getError(stateErr)
	}
	preferTCPBool := preferTCP == 1
	config, configType, configErr := state.GetConfigInstituteAccess(C.GoString(url), preferTCPBool)
	return C.CString(config), C.CString(configType), getError(configErr)
}

//export GetConfigCustomServer
func GetConfigCustomServer(
	name *C.char,
	url *C.char,
	preferTCP C.int,
) (*C.char, *C.char, *C.error) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, nil, getError(stateErr)
	}
	preferTCPBool := preferTCP == 1
	config, configType, configErr := state.GetConfigCustomServer(C.GoString(url), preferTCPBool)
	return C.CString(config), C.CString(configType), getError(configErr)
}

//export SetProfileID
func SetProfileID(name *C.char, data *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	profileErr := state.SetProfileID(C.GoString(data))
	return getError(profileErr)
}

//export ChangeSecureLocation
func ChangeSecureLocation(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	locationErr := state.ChangeSecureLocation()
	return getError(locationErr)
}

//export SetSecureLocation
func SetSecureLocation(name *C.char, data *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	locationErr := state.SetSecureLocation(C.GoString(data))
	return getError(locationErr)
}

//export GoBack
func GoBack(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	goBackErr := state.GoBack()
	return getError(goBackErr)
}

//export SetSearchServer
func SetSearchServer(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	setSearchErr := state.SetSearchServer()
	return getError(setSearchErr)
}

//export SetDisconnected
func SetDisconnected(name *C.char, cleanup C.int) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	setDisconnectedErr := state.SetDisconnected(int(cleanup) == 1)
	return getError(setDisconnectedErr)
}

//export SetDisconnecting
func SetDisconnecting(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	setDisconnectingErr := state.SetDisconnecting()
	return getError(setDisconnectingErr)
}

//export SetConnecting
func SetConnecting(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	setConnectingErr := state.SetConnecting()
	return getError(setConnectingErr)
}

//export SetConnected
func SetConnected(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	setConnectedErr := state.SetConnected()
	return getError(setConnectedErr)
}

//export RenewSession
func RenewSession(name *C.char) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	renewSessionErr := state.RenewSession()
	return getError(renewSessionErr)
}

//export ShouldRenewButton
func ShouldRenewButton(name *C.char) C.int {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.int(0)
	}
	shouldRenewBool := state.ShouldRenewButton()
	if shouldRenewBool {
		return C.int(1)
	}
	return C.int(0)
}

//export InFSMState
func InFSMState(name *C.char, checkState C.int) C.int {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.int(0)
	}
	inStateBool := state.InFSMState(client.FSMStateID(checkState))
	if inStateBool {
		return C.int(1)
	}
	return C.int(0)
}

//export SetSupportWireguard
func SetSupportWireguard(name *C.char, support C.int) *C.error {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return getError(stateErr)
	}
	state.SupportsWireguard = support == 1
	return nil
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
