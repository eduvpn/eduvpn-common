package eduvpn

import (
	"fmt"
)

type DiscoFileError struct {
	URL string
	Err error
}

func (e *DiscoFileError) Error() string {
	return fmt.Sprintf("failed obtaining disco file %s with error %v", e.URL, e.Err)
}

type DiscoSigFileError struct {
	URL string
	Err error
}

func (e *DiscoSigFileError) Error() string {
	return fmt.Sprintf("failed obtaining disco signature file %s with error %v", e.URL, e.Err)
}

type DiscoVerifyError struct {
	File    string
	Sigfile string
	Err     error
}

func (e *DiscoVerifyError) Error() string {
	return fmt.Sprintf("failed verifying file %s with signature %s due to error %v", e.File, e.Sigfile, e.Err)
}

// Helper function that gets a disco json
func getDiscoFile(jsonFile string) (string, error) {
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	fileBody, fileErr := HTTPGet(fileURL)

	if fileErr != nil {
		return "", &DiscoFileError{fileURL, fileErr}
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	sigBody, sigFileErr := HTTPGet(sigURL)

	if sigFileErr != nil {
		return "", &DiscoSigFileError{URL: sigURL, Err: sigFileErr}
	}

	// Verify signature
	// TODO: Handle this by keeping track of the previous sign time
	// Wrappers must do this?
	var previousSigTime uint64 = 0
	forcePrehash := false
	verifySuccess, verifyErr := Verify(string(sigBody), fileBody, jsonFile, previousSigTime, forcePrehash)

	if !verifySuccess || verifyErr != nil {
		return "", &DiscoVerifyError{File: jsonFile, Sigfile: sigFile, Err: verifyErr}
	}

	return string(fileBody), nil
}

type GetListError struct {
	File string
	Err  error
}

func (e *GetListError) Error() string {
	return fmt.Sprintf("failed getting disco list file %s with error %v", e.File, e.Err)
}

// Get the organization list
func GetOrganizationsList() (string, error) {
	file := "organization_list.json"
	body, err := getDiscoFile(file)
	if err != nil {
		return "", &GetListError{File: file, Err: err}
	}
	return body, nil
}

// Get the server list
func GetServersList() (string, error) {
	file := "server_list.json"
	body, err := getDiscoFile("server_list.json")
	if err != nil {
		return "", &GetListError{File: file, Err: err}
	}
	return body, nil
}
