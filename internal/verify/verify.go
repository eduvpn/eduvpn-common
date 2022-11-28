// package verify implement signature verification using minisign
package verify

import (
	"fmt"

	"github.com/eduvpn/eduvpn-common/types"
	"github.com/jedisct1/go-minisign"
)

// Verify verifies the signature (.minisig file format) on signedJSON.
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
func Verify(
	signatureFileContent string,
	signedJSON []byte,
	expectedFileName string,
	minSignTime uint64,
	forcePrehash bool,
) (bool, error) {
	// keys taken from https://git.sr.ht/~eduvpn/disco.eduvpn.org#public-keys
	keyStrs := []string{
		"RWRtBSX1alxyGX+Xn3LuZnWUT0w//B6EmTJvgaAxBMYzlQeI+jdrO6KF", // fkooman@tuxed.net, kolla@uninett.no
		"RWQKqtqvd0R7rUDp0rWzbtYPA3towPWcLDCl7eY9pBMMI/ohCmrS0WiM", // RoSp
	}
	valid, err := verifyWithKeys(
		signatureFileContent,
		signedJSON,
		expectedFileName,
		minSignTime,
		keyStrs,
		forcePrehash,
	)
	if err != nil {
		return valid, types.NewWrappedError("failed signature verify", err)
	}
	return valid, nil
}

// verifyWithKeys verifies the Minisign signature in signatureFileContent (minisig file format) over the server_list/organization_list JSON in signedJSON.
//
// Verification is performed using a matching key in allowedPublicKeys.
// The signature is checked to be a Ed25519 Minisign (optionally Ed25519 Blake2b-512 prehashed, see forcePrehash) signature with a valid trusted comment.
// The file type that is verified is indicated by expectedFileName, which must be one of "server_list.json"/"organization_list.json".
// The trusted comment is checked to be of the form "timestamp:<timestamp>\tfile:<expectedFileName>", optionally suffixed by something, e.g. "\thashed".
// The signature is checked to have a timestamp with a value of at least minSignTime, which is a UNIX timestamp without milliseconds.
//
// The return value will either be (true, nil) on success or (false, detailedVerifyError) on failure.
// Note that every error path is wrapped in a custom type here because minisign does not return custom error types, they use errors.New.
func verifyWithKeys(
	signatureFileContent string,
	signedJSON []byte,
	filename string,
	minSignTime uint64,
	allowedPublicKeys []string,
	forcePrehash bool,
) (bool, error) {
	switch filename {
	case "server_list.json", "organization_list.json":
		break
	default:
		return false, &UnknownExpectedFilenameError{
			Filename: filename,
			Expected: "server_list.json or organization_list.json",
		}
	}

	sig, err := minisign.DecodeSignature(signatureFileContent)
	if err != nil {
		return false, &InvalidSignatureFormatError{Err: err}
	}

	// Check if signature is prehashed, see https://jedisct1.github.io/minisign/#signature-format
	if forcePrehash && sig.SignatureAlgorithm != [2]byte{'E', 'D'} {
		return false, &InvalidSignatureAlgorithmError{
			Algorithm:       string(sig.SignatureAlgorithm[:]),
			WantedAlgorithm: "ED (BLAKE2b-prehashed EdDSA)",
		}
	}

	// Find allowed key used for signature
	for _, keyStr := range allowedPublicKeys {
		key, err := minisign.NewPublicKey(keyStr)
		if err != nil {
			// Should only happen if Verify is wrong or extraKey is invalid
			return false, &CreatePublicKeyError{PublicKey: keyStr, Err: err}
		}

		if sig.KeyId != key.KeyId {
			continue // Wrong key
		}

		valid, err := key.Verify(signedJSON, sig)
		if !valid {
			return false, &InvalidSignatureError{Err: err}
		}

		// Parse trusted comment
		var signTime uint64
		var sigFileName string
		// sigFileName cannot have spaces
		_, err = fmt.Sscanf(
			sig.TrustedComment,
			"trusted comment: timestamp:%d\tfile:%s",
			&signTime,
			&sigFileName,
		)
		if err != nil {
			return false, &InvalidTrustedCommentError{
				TrustedComment: sig.TrustedComment,
				Err:            err,
			}
		}

		if sigFileName != filename {
			return false, &WrongSigFilenameError{Filename: filename, SigFilename: sigFileName}
		}

		if signTime < minSignTime {
			return false, &SigTimeEarlierError{SigTime: signTime, MinSigTime: minSignTime}
		}

		return true, nil
	}

	// No matching allowed key found
	return false, &UnknownKeyError{Filename: filename}
}

type UnknownExpectedFilenameError struct {
	Filename string
	Expected string
}

func (e *UnknownExpectedFilenameError) Error() string {
	return fmt.Sprintf("invalid filename: %s, expected: %s", e.Filename, e.Expected)
}

type InvalidSignatureFormatError struct {
	Err error
}

func (e *InvalidSignatureFormatError) Error() string {
	return fmt.Sprintf("invalid signature format with error: %v", e.Err)
}

func (e *InvalidSignatureFormatError) Unwrap() error {
	return e.Err
}

type InvalidSignatureAlgorithmError struct {
	Algorithm       string
	WantedAlgorithm string
}

func (e *InvalidSignatureAlgorithmError) Error() string {
	return fmt.Sprintf(
		"invalid signature algorithm: %s, wanted: %s",
		e.Algorithm,
		e.WantedAlgorithm,
	)
}

type CreatePublicKeyError struct {
	PublicKey string
	Err       error
}

func (e *CreatePublicKeyError) Error() string {
	return fmt.Sprintf("failed to create public key: %s with error: %v", e.PublicKey, e.Err)
}

func (e *CreatePublicKeyError) Unwrap() error {
	return e.Err
}

type InvalidSignatureError struct {
	Err error
}

func (e *InvalidSignatureError) Error() string {
	return fmt.Sprintf("invalid signature with error: %v", e.Err)
}

func (e *InvalidSignatureError) Unwrap() error {
	return e.Err
}

type InvalidTrustedCommentError struct {
	TrustedComment string
	Err            error
}

func (e *InvalidTrustedCommentError) Error() string {
	return fmt.Sprintf("invalid trusted comment: %s with error: %v", e.TrustedComment, e.Err)
}

func (e *InvalidTrustedCommentError) Unwrap() error {
	return e.Err
}

type WrongSigFilenameError struct {
	Filename    string
	SigFilename string
}

func (e *WrongSigFilenameError) Error() string {
	return fmt.Sprintf(
		"wrong filename: %s, expected filename: %s for signature",
		e.Filename,
		e.SigFilename,
	)
}

type SigTimeEarlierError struct {
	SigTime    uint64
	MinSigTime uint64
}

func (e *SigTimeEarlierError) Error() string {
	return fmt.Sprintf("Sign time: %d is earlier than sign time: %d", e.SigTime, e.MinSigTime)
}

type UnknownKeyError struct {
	Filename string
}

func (e *UnknownKeyError) Error() string {
	return fmt.Sprintf("signature for filename: %s was created with an unknown key", e.Filename)
}
