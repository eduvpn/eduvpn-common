package main

import "C"

import "eduvpn-common"

// Functions here should not take string parameters, see https://pkg.go.dev/cmd/cgo#hdr-C_references_to_Go

// Verify verifies a signature on a JSON file. See eduvpn_verify.Verify for more details.
// It returns 0 for a valid signature and a nonzero eduvpn_verify.VerifyErrorCode otherwise.
//export Verify
func Verify(signatureFileContent []byte, signedJson []byte, expectedFileName []byte, minSignTime uint64) int {
	valid, err := eduvpn_verify.Verify(string(signatureFileContent), signedJson, string(expectedFileName), minSignTime)
	if valid {
		return 0
	} else {
		return int(err.(eduvpn_verify.VerifyError).Code)
	}
}

// InsecureTestingSetExtraKey adds an extra allowed key for verification with Verify.
// ONLY USE FOR TESTING. Not Thread-safe. Do not call in parallel to Verify.
//export InsecureTestingSetExtraKey
func InsecureTestingSetExtraKey(keyString []byte) {
	eduvpn_verify.InsecureTestingSetExtraKey(string(keyString))
}

func main() { panic("compile with -buildmode=c-shared") }
