package eduvpn

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func getFileUrl(url string) ([]byte, error) {
	// Do a Get request to the specified url
	resp, reqErr := http.Get(url)
	if reqErr != nil {
		return nil, detailedVPNError{errRequestFileError, fmt.Sprintf("request failed for file url %s", url), reqErr}
	}
	// Close the response body at the end
	defer resp.Body.Close()

	// Check if http response code is ok
	if resp.StatusCode != http.StatusOK {
		return nil, detailedVPNError{errRequestFileHTTPError, fmt.Sprintf("http status not ok for file url %s", url), nil}
	}
	// Read the body
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return nil, detailedVPNError{errRequestFileReadError, fmt.Sprintf("error reading body from file url %s", url), readErr}
	}
	return body, nil
}

// Helper function that gets a disco json
// TODO: Verify signature
func getDiscoFile(jsonFile string) (string, error) {
	// Get json data
	fileUrl := "https://disco.eduvpn.org/v2/" + jsonFile
	fileBody, error := getFileUrl(fileUrl)

	if error != nil {
		return "", error
	}

	// Get signature
	sigUrl := fileUrl + ".minisig"
	sigBody, error := getFileUrl(sigUrl)

	if error != nil {
		return "", error
	}

	// Verify signature
	// TODO: Handle this by keeping track of the previous sign time
	// Wrappers must do this?
	var previousSigTime uint64 = 0
	forcePrehash := false
	verifySuccess, verifyErr := Verify(string(sigBody), fileBody, jsonFile, previousSigTime, forcePrehash)

	if !verifySuccess || verifyErr != nil {
		return "", detailedVPNError{errVerifySigError, "Signature is not valid", verifyErr}
	}

	return string(fileBody), nil
}

// Get the organization list
func GetOrganizationsList() (string, error) {
	body, err := getDiscoFile("organization_list.json")
	if err != nil {
		return "", err.(detailedRequestError).ToRequestError()
	}
	return body, nil
}

// Get the server list
func GetServersList() (string, error) {
	return getDiscoFile("server_list.json")
}

// RequestErrorCode Simplified error code for public interface.
type RequestErrorCode = VPNErrorCode
type RequestError = VPNError

// detailedRequestErrorCode used for unit tests.
type detailedRequestErrorCode = detailedVPNErrorCode
type detailedRequestError = detailedVPNError

const (
	ErrRequestFileError RequestErrorCode = iota + 1
	ErrVerifySigError
)

const (
	errRequestFileError detailedRequestErrorCode = iota + 1
	errRequestFileHTTPError
	errRequestFileReadError
	errVerifySigError
)

func (err detailedRequestError) ToRequestError() RequestError {
	return RequestError{err.Code.ToRequestErrorCode(), err}
}

func (code detailedRequestErrorCode) ToRequestErrorCode() RequestErrorCode {
	switch code {
	case errRequestFileError:
	case errRequestFileReadError:
	case errRequestFileHTTPError:
		return ErrRequestFileError
	case errVerifySigError:
		return ErrVerifySigError
	}
	panic("invalid detailedRequestErrorCode")
}
