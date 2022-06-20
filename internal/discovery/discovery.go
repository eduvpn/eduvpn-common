package discovery

import (
	"encoding/json"
	"fmt"

	"github.com/jwijenbergh/eduvpn-common/internal/fsm"
	"github.com/jwijenbergh/eduvpn-common/internal/http"
	"github.com/jwijenbergh/eduvpn-common/internal/log"
	"github.com/jwijenbergh/eduvpn-common/internal/types"
	"github.com/jwijenbergh/eduvpn-common/internal/util"
	"github.com/jwijenbergh/eduvpn-common/internal/verify"
)

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
	FSM           *fsm.FSM
	Logger        *log.FileLogger
}

// Helper function that gets a disco json
func getDiscoFile(jsonFile string, previousVersion uint64, structure interface{}) error {
	errorMessage := fmt.Sprintf("failed getting file: %s from the Discovery server", jsonFile)
	// Get json data
	discoURL := "https://disco.eduvpn.org/v2/"
	fileURL := discoURL + jsonFile
	_, fileBody, fileErr := http.HTTPGet(fileURL)

	if fileErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: fileErr}
	}

	// Get signature
	sigFile := jsonFile + ".minisig"
	sigURL := discoURL + sigFile
	_, sigBody, sigFileErr := http.HTTPGet(sigURL)

	if sigFileErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: sigFileErr}
	}

	// Verify signature
	// Set this to true when we want to force prehash
	forcePrehash := false
	verifySuccess, verifyErr := verify.Verify(string(sigBody), fileBody, jsonFile, previousVersion, forcePrehash)

	if !verifySuccess || verifyErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: verifyErr}
	}

	// Parse JSON to extract version and list
	jsonErr := json.Unmarshal(fileBody, structure)

	if jsonErr != nil {
		return &types.WrappedErrorMessage{Message: errorMessage, Err: jsonErr}
	}

	return nil
}

func (discovery *Discovery) Init(fsm *fsm.FSM, logger *log.FileLogger) {
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
	now := util.GenerateTimeSeconds()
	if now >= should_update_time {
		return true
	}
	discovery.Logger.Log(log.LOG_INFO, "No update needed for servers, 1h is not passed yet")
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
		return string(discovery.Organizations.JSON), &types.WrappedErrorMessage{Message: "failed getting organizations in Discovery", Err: err}
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
		return string(discovery.Servers.JSON), &types.WrappedErrorMessage{Message: "failed getting servers in Discovery", Err: err}
	}
	// Update servers timestamp
	discovery.Servers.Timestamp = util.GenerateTimeSeconds()
	return string(discovery.Servers.JSON), nil
}
