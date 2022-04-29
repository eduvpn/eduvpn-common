package internal

import (
	"encoding/json"
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

type DiscoJSONError struct {
	Body string
	Err  error
}

func (e *DiscoJSONError) Error() string {
	return fmt.Sprintf("failed parsing JSON for contents %s with error %v", e.Body, e.Err)
}

type OrganizationList struct {
	JSON      json.RawMessage `json:"organization_list"`
	Version   uint64          `json:"v"`
	Timestamp int64           `json:"-"`
}

type ServersList struct {
	JSON      json.RawMessage `json:"server_list"`
	Version   uint64          `json:"v"`
	Timestamp int64           `json:"-"`
}

type Discovery struct {
	Organizations OrganizationList
	Servers       ServersList
	FSM           *FSM
	Logger        *FileLogger
}

// Helper function that gets a disco json
func getDiscoFile(jsonFile string, previousVersion uint64, structure interface{}) error {
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	_, fileBody, fileErr := HTTPGet(fileURL)

	if fileErr != nil {
		return &DiscoFileError{fileURL, fileErr}
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	_, sigBody, sigFileErr := HTTPGet(sigURL)

	if sigFileErr != nil {
		return &DiscoSigFileError{URL: sigURL, Err: sigFileErr}
	}

	// Verify signature
	// Set this to true when we want to force prehash
	forcePrehash := false
	verifySuccess, verifyErr := Verify(string(sigBody), fileBody, jsonFile, previousVersion, forcePrehash)

	if !verifySuccess || verifyErr != nil {
		return &DiscoVerifyError{File: jsonFile, Sigfile: sigFile, Err: verifyErr}
	}

	// Parse JSON to extract version and list
	jsonErr := json.Unmarshal(fileBody, structure)

	if jsonErr != nil {
		return &DiscoJSONError{Body: string(fileBody), Err: jsonErr}
	}

	return nil
}

type GetListError struct {
	File string
	Err  error
}

func (e *GetListError) Error() string {
	return fmt.Sprintf("failed getting disco list file %s with error %v", e.File, e.Err)
}

func (discovery *Discovery) Init(fsm *FSM, logger *FileLogger) {
	discovery.FSM = fsm
	discovery.Logger = logger
}

// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [TODO] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [TODO] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
func (discovery *Discovery) DetermineOrganizationsUpdate() bool {
	return string(discovery.Organizations.JSON) == ""
}

// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [Implemented] The application MUST always fetch the server_list.json at application start.
// - The application MAY refresh the server_list.json periodically, e.g. once every hour.
func (discovery *Discovery) DetermineServersUpdate() bool {
	// No servers, we should update
	if string(discovery.Servers.JSON) == "" {
		return true
	}
	// 1 hour from the last update
	should_update_time := discovery.Servers.Timestamp + 3600
	now := GenerateTimeSeconds()
	if now >= should_update_time {
		return true
	}
	discovery.Logger.Log(LOG_INFO, "No update needed for servers, 1h is not passed yet")
	return false
}

// Get the organization list
func (discovery *Discovery) GetOrganizationsList() (string, error) {
	if !discovery.DetermineOrganizationsUpdate() {
		return string(discovery.Organizations.JSON), nil
	}
	file := "organization_list.json"
	err := getDiscoFile(file, discovery.Organizations.Version, &discovery.Organizations)
	if err != nil {
		// Return previous with an error
		return string(discovery.Organizations.JSON), &GetListError{File: file, Err: err}
	}
	return string(discovery.Organizations.JSON), nil
}

// Get the server list
func (discovery *Discovery) GetServersList() (string, error) {
	if !discovery.DetermineServersUpdate() {
		return string(discovery.Servers.JSON), nil
	}
	file := "server_list.json"
	err := getDiscoFile(file, discovery.Servers.Version, &discovery.Servers)
	if err != nil {
		// Return previous with an error
		return string(discovery.Servers.JSON), &GetListError{File: file, Err: err}
	}
	// Update servers timestamp
	discovery.Servers.Timestamp = GenerateTimeSeconds()
	return string(discovery.Servers.JSON), nil
}
