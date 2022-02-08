package eduvpn_discovery

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"fmt"
)

// Struct that defines the json format for
// url: "https://disco.eduvpn.org/v2/organization_list.json"
type organizations struct {
	v string `json:"v"`
	OrganizationList []struct {
		DisplayName struct {
			En string `json:"en"`
		} `json:"display_name"`
		OrgId string `json:"org_id"`
		SecureInternetHome string `json:"secure_internet_home"`
		KeywordList struct {
			En string `json:"en"`
		} `json:"keyword_list"`
	} `json:"organization_list"`
}

// Struct that defines the json format for
// url: "https://disco.eduvpn.org/v2/server_list.json"
type servers struct {
	v string `json:"v"`
	ServerList []struct {
		BaseUrl string `json:"base_url"`
		CountryCode string `json:"country_code"`
		PublicKeyList []string `json:"public_key_list"`
		ServerType string `json:"secure_internet"`
		SupportContact []string `json:"support_contact"`
	} `json:"server_list"`
}

// Helper function that gets a disco json
// TODO: Verify signature
func getDiscoJson(jsonFile string, structure interface{}) bool {
	url := "https://disco.eduvpn.org/v2/" + jsonFile
	// Do a Get request to the specified url
	resp, reqErr := http.Get(url)
	if reqErr != nil {
		fmt.Println("error making request")
		return false
	}

	// Read the body
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		fmt.Println("error reading body of request")
		return false
	}

	// Parse the json using the predefined struct
	error := json.Unmarshal([]byte(body), &structure)
	if error != nil {
		fmt.Println("error parsing server json")
		return false
	}
	return true
}

// Get the organization list
func getOrganizationList() bool {
	organizations := organizations{}
	return getDiscoJson("organization_list.json", &organizations)
}

// Get the server list
func getServerList() bool {
	servers := servers{}
	return getDiscoJson("server_list.json", &servers)
}
