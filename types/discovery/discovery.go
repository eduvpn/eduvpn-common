// Package discovery defines the public types that have to deal with discovery
package discovery

import "encoding/json"

// Organizations is the type that defines the upstream discovery format for the list of organizations
// It is a subset of the format from URL: "https://disco.eduvpn.org/v2/organization_list.json"
type Organizations struct {
	// List is the list/slice of organizations. Omitted if none are there
	List []Organization `json:"organization_list,omitempty"`
}

// Organization is the type that defines the upstream discovery format for a single organization
type Organization struct {
	// DisplayName is the map of strings from language tags to display names
	// Omitted if none is defined
	DisplayName MapOrString `json:"display_name,omitempty"`
	// OrgID is the organization ID for the server
	OrgID string `json:"org_id"`
	// score is the score internally used for sorting
	Score int `json:"-"`
}

// Servers is the type that defines the upstream discovery format for the list of servers
// url: "https://disco.eduvpn.org/v2/server_list.json"
type Servers struct {
	// List is the actual list of servers, omitted from the JSON if empty
	List []Server `json:"server_list,omitempty"`
}

// Server is a signle discovery server
type Server struct {
	// BaseURL is the base URL of the server which is used as an identifier for the server by the Go library
	BaseURL string `json:"base_url"`
	// DisplayName is the display name of the server, omitted if empty
	DisplayName MapOrString `json:"display_name,omitempty"`
	// Type is the type of the server, "secure_internet" or "institute_access"
	Type string `json:"server_type"`
	// CountryCode is the country code of the server if Type is "secure_internet", e.g. nl
	CountryCode string `json:"country_code"`
	// score is the score internally used for sorting
	Score int `json:"-"`
}

// MapOrString is a custom type as the upstream discovery format is a map or a value.
// This library always marshals the data as a map and then makes sure unmarshalling also gives a map
type MapOrString map[string]string

// UnmarshalJSON unmarshals the display name. It can either be a map or a string in the server list
// Unmarshal it by first trying a string and then the map.
func (displayName *MapOrString) UnmarshalJSON(data []byte) error {
	var displayNameString string

	err := json.Unmarshal(data, &displayNameString)

	if err == nil {
		*displayName = map[string]string{"en": displayNameString}
		return nil
	}

	var resultingMap map[string]string

	err = json.Unmarshal(data, &resultingMap)

	if err == nil {
		*displayName = resultingMap
		return nil
	}
	return err
}
