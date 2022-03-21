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
func Register(name *C.char, config_directory *C.char, stateCallback C.PythonCB) {
	P_StateCallback = stateCallback
	eduvpn.Register(eduvpn.GetVPNState(), C.GoString(name), C.GoString(config_directory), StateCallback)
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
