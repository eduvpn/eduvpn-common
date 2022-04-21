package eduvpn

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

type DiscoLists struct {
	Organizations OrganizationList
	Servers       ServersList
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

// FIXME: Implement based on
// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [IMPLEMENTED] on "first launch" when offering the search for "Institute Access" and "Organizations";
// - [TODO] when the user tries to add new server AND the user did NOT yet choose an organization before;
// - [TODO] when the authorization for the server associated with an already chosen organization is triggered, e.g. after expiry or revocation.
func (eduvpn *VPNState) DetermineOrganizationsUpdate() bool {
	return string(eduvpn.DiscoList.Organizations.JSON) == ""
}

// https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md
// - [Implemented] The application MUST always fetch the server_list.json at application start.
// - The application MAY refresh the server_list.json periodically, e.g. once every hour.
func (eduvpn *VPNState) DetermineServersUpdate() bool {
	// No servers, we should update
	if string(eduvpn.DiscoList.Servers.JSON) == "" {
		return true
	}
	// 1 hour from the last update
	should_update_time := eduvpn.DiscoList.Servers.Timestamp + 3600
	now := GenerateTimeSeconds()
	if now >= should_update_time {
		return true
	}
	GetVPNState().Log(LOG_INFO, "No update needed for servers, 1h is not passed yet")
	return false
}

// Get the organization list
func (eduvpn *VPNState) GetOrganizationsList() (string, error) {
	if !eduvpn.DetermineOrganizationsUpdate() {
		return string(eduvpn.DiscoList.Organizations.JSON), nil
	}
	file := "organization_list.json"
	err := getDiscoFile(file, eduvpn.DiscoList.Organizations.Version, &eduvpn.DiscoList.Organizations)
	if err != nil {
		// Return previous with an error
		return string(eduvpn.DiscoList.Organizations.JSON), &GetListError{File: file, Err: err}
	}
	return string(eduvpn.DiscoList.Organizations.JSON), nil
}

// Get the server list
func (eduvpn *VPNState) GetServersList() (string, error) {
	if !eduvpn.DetermineServersUpdate() {
		return string(eduvpn.DiscoList.Servers.JSON), nil
	}
	file := "server_list.json"
	err := getDiscoFile(file, eduvpn.DiscoList.Servers.Version, &eduvpn.DiscoList.Servers)
	if err != nil {
		// Return previous with an error
		return string(eduvpn.DiscoList.Servers.JSON), &GetListError{File: file, Err: err}
	}
	// Update servers timestamp
	eduvpn.DiscoList.Servers.Timestamp = GenerateTimeSeconds()
	return string(eduvpn.DiscoList.Servers.JSON), nil
}
