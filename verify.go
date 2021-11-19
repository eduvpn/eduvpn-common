package eduvpn_verify

import (
	"encoding/json"
	"fmt"
	"github.com/jedisct1/go-minisign"
)

// Verify verifies the signature (.minisig file format) on signedJson.
//
// expectedFileName must be set to the file type to be verified, either server_list.json or organization_list.json.
// minSign must be set to the minimum UNIX timestamp (without milliseconds) for the file version.
// This value should not be smaller than the time on the previous document verified.
//
// The return value will either be (true, nil) for a valid signature or (false, err) otherwise.
//
// Verify is a wrapper around verifyWithKeys where allowedPublicKeys is set to the list from https://git.sr.ht/~eduvpn/disco.eduvpn.org#public-keys.
func Verify(signatureFileContent string, signedJson []byte, expectedFileName string, minSignTime uint64) (bool, error) {
	keyStrs := []string{
		"RWRtBSX1alxyGX+Xn3LuZnWUT0w//B6EmTJvgaAxBMYzlQeI+jdrO6KF", // fkooman@tuxed.net, kolla@uninett.no
		"RWQKqtqvd0R7rUDp0rWzbtYPA3towPWcLDCl7eY9pBMMI/ohCmrS0WiM", // RoSp
	}
	valid, err := verifyWithKeys(signatureFileContent, signedJson, expectedFileName, minSignTime, keyStrs)
	if err != nil && err.(VerifyError).Code == ErrInvalidPublicKey {
		panic(err) // This should not happen
	}
	return valid, err
}

// verifyWithKeys verifies the Minisign signature in signatureFileContent (minisig file format) over the server_list/organization_list JSON in signedJson (UTF-8).
//
// Verification is performed using a matching key in allowedPublicKeys.
// The signature is checked to be a Blake2b-prehashed Ed25519 Minisign signature with a valid trusted comment.
// The file type that is verified is indicated by expectedFileName, which must be one of server_list.json/organization_list.json.
// The trusted comment is checked to be of the form "time:<timestamp>\tfile:<expectedFileName>", optionally suffixed by "\thashed".
// The JSON file and signature are checked to have a timestamp with a value of at least minSignTime, which is a UNIX timestamp without milliseconds;
// more precisely: min sign time <= sign time from trusted comment <= time from JSON 'v' tag.
// The JSON file is checked to be valid JSON and contain a tag with key server_list/organization_list, depending on expectedFileName.
//
// The return value will either be (true, nil) on success or (false, err) on failure.
func verifyWithKeys(signatureFileContent string, signedJson []byte, expectedFileName string, minSignTime uint64, allowedPublicKeys []string) (bool, error) {
	switch expectedFileName {
	case "server_list.json", "organization_list.json":
		break
	default:
		return false, VerifyError{ErrUnknownExpectedFileName,
			fmt.Sprintf("invalid expected file name (%v)", expectedFileName), nil}
	}

	sig, err := minisign.DecodeSignature(signatureFileContent)
	if err != nil {
		return false, VerifyError{ErrInvalidSignatureFormat, "invalid signature format", err}
	}

	if sig.SignatureAlgorithm != [2]byte{'E', 'D'} {
		return false, VerifyError{ErrInvalidSignatureAlgorithm, "BLAKE2b-prehashed EdDSA signature required", nil}
	}

	keys := make([]minisign.PublicKey, len(allowedPublicKeys))
	for i, keyStr := range allowedPublicKeys {
		key, err := minisign.NewPublicKey(keyStr)
		if err != nil {
			return false, VerifyError{ErrInvalidPublicKey, "internal error: could not create public key", err}
		}
		keys[i] = key
	}

	for _, key := range keys {
		if sig.KeyId != key.KeyId {
			continue
		}

		valid, err := key.Verify(signedJson, sig)
		if !valid {
			return false, VerifyError{ErrInvalidSignature, "invalid signature", err}
		}

		var signTime uint64
		var sigFileName string
		// sigFileName cannot have spaces
		_, err = fmt.Sscanf(sig.TrustedComment, "trusted comment: time:%d\tfile:%s", &signTime, &sigFileName)
		if err != nil {
			return false, VerifyError{ErrInvalidTrustedComment,
				fmt.Sprintf("failed to interpret trusted comment (%q)", sig.TrustedComment), err}
		}

		if sigFileName != expectedFileName {
			return false, VerifyError{ErrWrongFileName,
				fmt.Sprintf("signature was on file %q instead of expected %q", sigFileName, expectedFileName), nil}
		}

		// Technically redundant due to checks below
		if signTime < minSignTime {
			return false, VerifyError{ErrTooOld,
				fmt.Sprintf("signature was created at %v < minimum time (%v)", signTime, minSignTime), nil}
		}

		var signedData struct {
			Time             uint64      `json:"v"`
			ServerList       interface{} `json:"server_list"`
			OrganizationList interface{} `json:"organization_list"`
		}
		err = json.Unmarshal(signedJson, &signedData)
		if err != nil {
			return false, VerifyError{ErrWrongFileContent, "failed to parse JSON", err}
		}

		if signedData.Time == 0 {
			// Field absent or 0
			return false, VerifyError{ErrWrongFileContent, "JSON file must have nonzero 'v' field", nil}
		}

		if signedData.Time > signTime {
			return false, VerifyError{ErrWrongFileContent, fmt.Sprintf(
				"list was created at %v > signature time (%v), which should be impossible",
				signedData.Time, signTime), nil}
		}

		if signedData.Time < minSignTime {
			return false, VerifyError{ErrTooOld,
				fmt.Sprintf("list was created at %v < minimum time (%v)", signedData.Time, minSignTime), nil}
		}

		switch expectedFileName {
		case "server_list.json":
			if _, isServerList := signedData.ServerList.([]interface{}); !isServerList {
				return false, VerifyError{ErrWrongFileContent, "JSON file does not have a server_list", nil}
			}
		case "organization_list.json":
			if _, isOrganizationList := signedData.OrganizationList.([]interface{}); !isOrganizationList {
				return false, VerifyError{ErrWrongFileContent, "JSON file does not have an organization_list", nil}
			}
		}

		return true, nil
	}

	return false, VerifyError{ErrWrongKey, "signature was created with an unknown key", nil}
}

type VerifyErrCode int

const (
	ErrUnknownExpectedFileName VerifyErrCode = iota
	ErrInvalidSignatureFormat
	ErrInvalidSignatureAlgorithm
	ErrInvalidPublicKey
	ErrInvalidSignature
	ErrInvalidTrustedComment
	ErrWrongFileName
	ErrWrongFileContent
	ErrTooOld
	ErrWrongKey
)

type VerifyError struct {
	Code    VerifyErrCode
	Message string
	Cause   error
}

func (err VerifyError) Error() string {
	return err.Message
}
func (err VerifyError) Unwrap() error {
	return err.Cause
}
