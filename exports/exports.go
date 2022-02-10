package main

import "C"

import "github.com/jwijenbergh/eduvpn-common/src"

// Functions here should probably not take string parameters, see https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go

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
