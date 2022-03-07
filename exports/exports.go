package main

/*
#include <stdlib.h>
*/
import "C"
import "unsafe"

import "github.com/jwijenbergh/eduvpn-common/src"

// Functions here should probably not take string parameters, see https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go
// GetOrganizationsList gets the list of organizations from the disco server.
// Returns the json data as a string and an error code. This is used as key for looking up data.
//export GetOrganizationsList
func GetOrganizationsList() (*C.char, int8) {
	body, err := eduvpn.GetOrganizationsList()
	if err != nil {
		return nil, int8(err.(eduvpn.RequestError).Code)
	}
	return C.CString(body), 0
}

//export Register
func Register(name *C.char, url *C.char) {
	eduvpn.Register(eduvpn.GetVPNState(), C.GoString(name), C.GoString(url))
}

//export InitializeOAuth
func InitializeOAuth() (*C.char) {
	url, err := eduvpn.InitializeOAuth(eduvpn.GetVPNState())
	if err != nil {
		panic(err)
	}
	return C.CString(url)
}

// GetServersList gets the list of servers from the disco server.
// Returns the json data as a string and an error code. This is used as key for looking up data.
//export GetServersList
func GetServersList() (*C.char, int8) {
	body, err := eduvpn.GetServersList()
	if err != nil {
		return nil, int8(err.(eduvpn.RequestError).Code)
	}
	return C.CString(body), 0
}

//export FreeString
func FreeString(addr *C.char) {
	C.free(unsafe.Pointer(addr))
}
//
//// GetServerList gets the list of servers from the disco server.
//// Returns the unix timestamp of the data. This is used as key for looking up data.
//func GetServersList() ([]byte, int8) {
//	body, err := eduvpn.GetServersList()
//	if err != nil {
//		return nil, int8(err.(eduvpn.RequestError).Code)
//	}
//	return body, 0
//}

// Verify verifies a signature on a JSON file. See eduvpn.Verify for more details.
// It returns 0 for a valid signature and a nonzero eduvpn.VerifyErrorCode otherwise.
// signatureFileContent must be UTF-8-encoded.
//export Verify
func Verify(signatureFileContent []byte, signedJson []byte, expectedFileName []byte, minSignTime uint64) int8 {
	valid, err := eduvpn.Verify(string(signatureFileContent), signedJson, string(expectedFileName), minSignTime, false)
	if valid {
		return 0
	} else {
		return int8(err.(eduvpn.VerifyError).Code)
	}
}

// InsecureTestingSetExtraKey adds an extra allowed key for verification with Verify.
// ONLY USE FOR TESTING. Not Thread-safe. Do not call in parallel to Verify.
// keyString must be an ASCII Base64-encoded key.
//export InsecureTestingSetExtraKey
func InsecureTestingSetExtraKey(keyString []byte) {
	eduvpn.InsecureTestingSetExtraKey(string(keyString))
}

// Not used in library, but needed to compile.
func main() { panic("compile with -buildmode=c-shared") }
