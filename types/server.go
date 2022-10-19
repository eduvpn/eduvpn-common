package types

import (
	"encoding/json"
	"time"
)

// Shared server types

// Structs that define the json format for
// url: "https://disco.eduvpn.org/v2/organization_list.json"
type DiscoveryOrganizations struct {
	Version   uint64                  `json:"v"`
	List      []DiscoveryOrganization `json:"organization_list"`
	Timestamp time.Time               `json:"go_timestamp"`
}

type DiscoveryOrganization struct {
	DisplayName        DiscoMapOrString `json:"display_name"`
	OrgId              string           `json:"org_id"`
	SecureInternetHome string           `json:"secure_internet_home"`
	KeywordList        DiscoMapOrString `json:"keyword_list"`
}

// Structs that define the json format for
// url: "https://disco.eduvpn.org/v2/server_list.json"
type DiscoveryServers struct {
	Version   uint64            `json:"v"`
	List      []DiscoveryServer `json:"server_list"`
	Timestamp time.Time         `json:"go_timestamp"`
}

type DiscoMapOrString map[string]string

// The display name can either be a map or a string in the server list
// Unmarshal it by first trying a string and then the map
func (DN *DiscoMapOrString) UnmarshalJSON(data []byte) error {
	var displayNameString string

	err := json.Unmarshal(data, &displayNameString)

	if err == nil {
		*DN = map[string]string{"en": displayNameString}
		return nil
	}

	var resultingMap map[string]string

	err = json.Unmarshal(data, &resultingMap)

	if err == nil {
		*DN = resultingMap
		return nil
	}
	return err
}

type DiscoveryServer struct {
	AuthenticationURLTemplate string           `json:"authentication_url_template"`
	BaseURL                   string           `json:"base_url"`
	CountryCode               string           `json:"country_code"`
	DisplayName               DiscoMapOrString `json:"display_name,omitempty"`
	KeywordList               DiscoMapOrString `json:"keyword_list"`
	PublicKeyList             []string         `json:"public_key_list"`
	Type                      string           `json:"server_type"`
	SupportContact            []string         `json:"support_contact"`
}
