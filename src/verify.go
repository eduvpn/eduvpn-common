package eduvpn

import (
	"fmt"
	"github.com/jedisct1/go-minisign"
	"os"
)

// getKeys returns keys taken from https://git.sr.ht/~eduvpn/disco.eduvpn.org#public-keys.
func getKeys() []string {
	return []string{
		"RWRtBSX1alxyGX+Xn3LuZnWUT0w//B6EmTJvgaAxBMYzlQeI+jdrO6KF", // fkooman@tuxed.net, kolla@uninett.no
		"RWQKqtqvd0R7rUDp0rWzbtYPA3towPWcLDCl7eY9pBMMI/ohCmrS0WiM", // RoSp
	}
}

// Verify verifies the signature (.minisig file format) on signedJson.
//
// expectedFileName must be set to the file type to be verified, either "server_list.json" or "organization_list.json".
// minSign must be set to the minimum UNIX timestamp (without milliseconds) for the file version.
// This value should not be smaller than the time on the previous document verified.
// forcePrehash indicates whether or not we want to force the use of prehashed signatures
// In the future we want to remove this parameter and only allow prehashed signatures
//
// The return value will either be (true, nil) for a valid signature or (false, VerifyError) otherwise.
//
// Verify is a wrapper around verifyWithKeys where allowedPublicKeys is set to the list from https://git.sr.ht/~eduvpn/disco.eduvpn.org#public-keys.
func Verify(signatureFileContent string, signedJson []byte, expectedFileName string, minSignTime uint64, forcePrehash bool) (bool, error) {
	keyStrs := getKeys()
	if extraKey != "" {
		keyStrs = append(keyStrs, extraKey)
		_, err := fmt.Fprintf(os.Stderr, "INSECURE TEST MODE ENABLED WITH KEY %q\n", extraKey)
		if err != nil {
			panic(err)
		}
	}
	valid, err := verifyWithKeys(signatureFileContent, signedJson, expectedFileName, minSignTime, keyStrs, forcePrehash)
	if err != nil {
		if err.(detailedVerifyError).Code == errInvalidPublicKey {
			panic(err) // This should not happen unless keyStrs has an invalid key
		}
		return valid, err.(detailedVerifyError).ToVerifyError()
	}
	return valid, nil
}

// extraKey is an extra allowed key for testing.
var extraKey = ""

// InsecureTestingSetExtraKey adds an extra allowed key for verification with Verify.
// ONLY USE FOR TESTING. Applies to all threads. Probably not thread-safe. Do not call in parallel to Verify.
//
// keyString must be a Base64-encoded Minisign key, or empty to reset.
func InsecureTestingSetExtraKey(keyString string) {
	extraKey = keyString
}

// verifyWithKeys verifies the Minisign signature in signatureFileContent (minisig file format) over the server_list/organization_list JSON in signedJson.
//
// Verification is performed using a matching key in allowedPublicKeys.
// The signature is checked to be a Ed25519 Minisign (optionally Ed25519 Blake2b-512 prehashed, see forcePrehash) signature with a valid trusted comment.
// The file type that is verified is indicated by expectedFileName, which must be one of "server_list.json"/"organization_list.json".
// The trusted comment is checked to be of the form "timestamp:<timestamp>\tfile:<expectedFileName>", optionally suffixed by something, e.g. "\thashed".
// The signature is checked to have a timestamp with a value of at least minSignTime, which is a UNIX timestamp without milliseconds.
//
// The return value will either be (true, nil) on success or (false, detailedVerifyError) on failure.
func verifyWithKeys(signatureFileContent string, signedJson []byte, expectedFileName string, minSignTime uint64, allowedPublicKeys []string, forcePrehash bool) (bool, error) {
	switch expectedFileName {
	case "server_list.json", "organization_list.json":
		break
	default:
		return false, detailedVerifyError{errUnknownExpectedFileName, "invalid expected file name", nil}
	}

	sig, err := minisign.DecodeSignature(signatureFileContent)
	if err != nil {
		return false, detailedVerifyError{errInvalidSignatureFormat, "invalid signature format", err}
	}

	// Check if signature is prehashed, see https://jedisct1.github.io/minisign/#signature-format
	if forcePrehash && sig.SignatureAlgorithm != [2]byte{'E', 'D'} {
		return false, detailedVerifyError{errInvalidSignatureAlgorithm, "BLAKE2b-prehashed EdDSA signature required", nil}
	}

	// Find allowed key used for signature
	for _, keyStr := range allowedPublicKeys {
		key, err := minisign.NewPublicKey(keyStr)
		if err != nil {
			// Should only happen if Verify is wrong or extraKey is invalid
			return false, detailedVerifyError{errInvalidPublicKey, "internal error: could not create public key", err}
		}

		if sig.KeyId != key.KeyId {
			continue // Wrong key
		}

		valid, err := key.Verify(signedJson, sig)
		if !valid {
			return false, detailedVerifyError{errInvalidSignature, "invalid signature", err}
		}

		// Parse trusted comment
		var signTime uint64
		var sigFileName string
		// sigFileName cannot have spaces
		_, err = fmt.Sscanf(sig.TrustedComment, "trusted comment: timestamp:%d\tfile:%s", &signTime, &sigFileName)
		if err != nil {
			return false, detailedVerifyError{errInvalidTrustedComment, "failed to interpret trusted comment", err}
		}

		if sigFileName != expectedFileName {
			return false, detailedVerifyError{errWrongFileName, "signature was created for wrong file", nil}
		}

		if signTime < minSignTime {
			return false, detailedVerifyError{errTooOld, "signature was created a time earlier than the minimum time specified", nil}
		}

		return true, nil
	}

	// No matching allowed key found
	return false, detailedVerifyError{errWrongKey, "signature was created with an unknown key", nil}
}

// VerifyErrorCode Simplified error code for public interface.
type VerifyErrorCode = VPNErrorCode
type VerifyError = VPNError
// detailedVerifyErrorCode used for unit tests.
type detailedVerifyErrorCode = detailedVPNErrorCode
type detailedVerifyError = detailedVPNError


const (
	ErrUnknownExpectedFileName    VerifyErrorCode = iota + 1 // Unknown expected file name specified. The signature has not been verified.
	ErrInvalidSignature                                      // Signature is invalid (for the expected file type).
	ErrInvalidSignatureUnknownKey                            // Signature was created with an unknown key and has not been verified.
	ErrTooOld                                                // Signature timestamp smaller than specified minimum signing time (rollback).
)

const (
	errUnknownExpectedFileName detailedVerifyErrorCode = iota + 1
	errInvalidSignatureFormat
	errInvalidSignatureAlgorithm
	errInvalidPublicKey
	errInvalidSignature
	errInvalidTrustedComment
	errWrongFileName
	errTooOld
	errWrongKey
)

func (err detailedVerifyError) ToVerifyError() VerifyError {
	return VerifyError{err.Code.ToVerifyErrorCode(), err}
}

func (code detailedVerifyErrorCode) ToVerifyErrorCode() VerifyErrorCode {
	switch code {
	case errUnknownExpectedFileName:
		return ErrUnknownExpectedFileName
	case errInvalidSignatureFormat:
		return ErrInvalidSignature
	case errInvalidSignatureAlgorithm:
		return ErrInvalidSignature
	case errInvalidPublicKey:
		panic("errInvalidPublicKey cannot be converted to VerifyErrorCode")
	case errInvalidSignature:
		return ErrInvalidSignature
	case errInvalidTrustedComment:
		return ErrInvalidSignature
	case errWrongFileName:
		return ErrInvalidSignature
	case errTooOld:
		return ErrTooOld
	case errWrongKey:
		return ErrInvalidSignatureUnknownKey
	}
	panic("invalid detailedVerifyErrorCode")
}

