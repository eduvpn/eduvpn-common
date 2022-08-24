package main

/*
#include <stdlib.h>

typedef void (*PythonCB)(const char* name, int oldstate, int newstate, const char* data);

__attribute__((weak))
void call_callback(PythonCB callback, const char *name, int oldstate, int newstate, const char* data)
{
    callback(name, oldstate, newstate, data);
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"

	"github.com/jwijenbergh/eduvpn-common"
)

var P_StateCallbacks map[string]C.PythonCB

var VPNStates map[string]*eduvpn.VPNState

func StateCallback(
	name string,
	old_state eduvpn.StateID,
	new_state eduvpn.StateID,
	data interface{},
) {
	P_StateCallback, exists := P_StateCallbacks[name]
	if !exists || P_StateCallback == nil {
		return
	}
	name_c := C.CString(name)
	oldState_c := C.int(old_state)
	newState_c := C.int(new_state)
	data_json, jsonErr := json.Marshal(data)
	var dataJsonString string
	if jsonErr != nil {
		// TODO: How to handle error further? Log?
		dataJsonString = "{}"
	} else {
		dataJsonString = string(data_json)
	}
	data_c := C.CString(dataJsonString)
	C.call_callback(P_StateCallback, name_c, oldState_c, newState_c, data_c)
	C.free(unsafe.Pointer(name_c))
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
func Register(
	name *C.char,
	config_directory *C.char,
	stateCallback C.PythonCB,
	debug C.int,
) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		state = &eduvpn.VPNState{}
	}
	if VPNStates == nil {
		VPNStates = make(map[string]*eduvpn.VPNState)
	}
	if P_StateCallbacks == nil {
		P_StateCallbacks = make(map[string]C.PythonCB)
	}
	VPNStates[nameStr] = state
	P_StateCallbacks[nameStr] = stateCallback
	registerErr := state.Register(
		nameStr,
		C.GoString(config_directory),
		func(old eduvpn.StateID, new eduvpn.StateID, data interface{}) {
			StateCallback(nameStr, old, new, data)
		},
		debug != 0,
	)

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

	return eduvpn.GetErrorJSONString(error)
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

type configJSON struct {
	Config     string `json:"config"`
	ConfigType string `json:"config_type"`
}

func getConfigJSON(config string, configType string) *C.char {
	object := &configJSON{Config: config, ConfigType: configType}
	jsonBytes, jsonErr := json.Marshal(object)

	if jsonErr != nil {
		panic(jsonErr)
	}

	return C.CString(string(jsonBytes))
}

//export RemoveSecureInternet
func RemoveSecureInternet(name *C.char) (*C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	removeErr := state.RemoveSecureInternet()
	return C.CString(ErrorToString(removeErr))
}

//export RemoveInstituteAccess
func RemoveInstituteAccess(name *C.char, url *C.char) (*C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	removeErr := state.RemoveInstituteAccess(C.GoString(url))
	return C.CString(ErrorToString(removeErr))
}

//export RemoveCustomServer
func RemoveCustomServer(name *C.char, url *C.char) (*C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	removeErr := state.RemoveCustomServer(C.GoString(url))
	return C.CString(ErrorToString(removeErr))
}

//export GetConfigSecureInternet
func GetConfigSecureInternet(name *C.char, orgID *C.char, forceTCP C.int) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	forceTCPBool := forceTCP == 1
	config, configType, configErr := state.GetConfigSecureInternet(C.GoString(orgID), forceTCPBool)
	return getConfigJSON(config, configType), C.CString(ErrorToString(configErr))
}

//export GetConfigInstituteAccess
func GetConfigInstituteAccess(name *C.char, url *C.char, forceTCP C.int) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	forceTCPBool := forceTCP == 1
	config, configType, configErr := state.GetConfigInstituteAccess(C.GoString(url), forceTCPBool)
	return getConfigJSON(config, configType), C.CString(ErrorToString(configErr))
}

//export GetConfigCustomServer
func GetConfigCustomServer(name *C.char, url *C.char, forceTCP C.int) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	forceTCPBool := forceTCP == 1
	config, configType, configErr := state.GetConfigCustomServer(C.GoString(url), forceTCPBool)
	return getConfigJSON(config, configType), C.CString(ErrorToString(configErr))
}

//export GetDiscoOrganizations
func GetDiscoOrganizations(name *C.char) (*C.char, *C.char) {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return nil, C.CString(ErrorToString(stateErr))
	}
	organizations, organizationsErr := state.GetDiscoOrganizations()
	return C.CString(organizations), C.CString(ErrorToString(organizationsErr))
}

//export GetDiscoServers
func GetDiscoServers(name *C.char) (*C.char, *C.char) {
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

//export ChangeSecureLocation
func ChangeSecureLocation(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	locationErr := state.ChangeSecureLocation()
	return C.CString(ErrorToString(locationErr))
}

//export SetSecureLocation
func SetSecureLocation(name *C.char, data *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	locationErr := state.SetSecureLocation(C.GoString(data))
	return C.CString(ErrorToString(locationErr))
}

//export GoBack
func GoBack(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	goBackErr := state.GoBack()
	return C.CString(ErrorToString(goBackErr))
}

//export SetSearchServer
func SetSearchServer(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setSearchErr := state.SetSearchServer()
	return C.CString(ErrorToString(setSearchErr))
}

//export SetDisconnected
func SetDisconnected(name *C.char, cleanup C.int) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setDisconnectedErr := state.SetDisconnected(int(cleanup) == 1)
	return C.CString(ErrorToString(setDisconnectedErr))
}

//export SetDisconnecting
func SetDisconnecting(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setDisconnectingErr := state.SetDisconnecting()
	return C.CString(ErrorToString(setDisconnectingErr))
}

//export SetConnecting
func SetConnecting(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	setConnectingErr := state.SetConnecting()
	return C.CString(ErrorToString(setConnectingErr))
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

//export RenewSession
func RenewSession(name *C.char) *C.char {
	nameStr := C.GoString(name)
	state, stateErr := GetVPNState(nameStr)
	if stateErr != nil {
		return C.CString(ErrorToString(stateErr))
	}
	renewSessionErr := state.RenewSession()
	return C.CString(ErrorToString(renewSessionErr))
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
	inStateBool := state.InFSMState(eduvpn.StateID(checkState))
	if inStateBool {
		return C.int(1)
	}
	return C.int(0)
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
