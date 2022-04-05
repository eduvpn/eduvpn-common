package main

/*
#include <stdlib.h>

typedef void (*PythonCB)(const char* oldstate, const char* newstate, const char* data);

// FIXME: Remove this, see: https://stackoverflow.com/questions/58606884/multiple-definition-when-using-cgo
__attribute__((weak))
void call_callback(PythonCB callback, const char* oldstate, const char* newstate, const char* data)
{
    callback(oldstate, newstate, data);
}
*/
import "C"
import "unsafe"
import "github.com/jwijenbergh/eduvpn-common/src"

var P_StateCallback C.PythonCB

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

//export Register
func Register(name *C.char, config_directory *C.char, stateCallback C.PythonCB, debug C.int) *C.char {
	P_StateCallback = stateCallback
	state := eduvpn.GetVPNState()
	registerErr := state.Register(C.GoString(name), C.GoString(config_directory), StateCallback, debug != 0)
	return C.CString(ErrorToString(registerErr))
}

//export Deregister
func Deregister() {
	state := eduvpn.GetVPNState()
	state.Deregister()
}

func ErrorToString(error error) string {
	if error == nil {
		return ""
	}

	return error.Error()
}

//export Connect
func Connect(url *C.char) (*C.char, *C.char) {
	state := eduvpn.GetVPNState()
	config, configErr := state.Connect(C.GoString(url))
	return C.CString(config), C.CString(ErrorToString(configErr))
}

//export GetOrganizationsList
func GetOrganizationsList() (*C.char, *C.char) {
	state := eduvpn.GetVPNState()
	organizations, organizationsErr := state.GetOrganizationsList()
	return C.CString(organizations), C.CString(ErrorToString(organizationsErr))
}


//export GetServersList
func GetServersList() (*C.char, *C.char) {
	state := eduvpn.GetVPNState()
	servers, serversErr := state.GetServersList()
	return C.CString(servers), C.CString(ErrorToString(serversErr))
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
