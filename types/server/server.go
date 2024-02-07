// package server defines public types that have to deal with the VPN server
package server

import (
	"encoding/json"
	"fmt"

	"github.com/eduvpn/eduvpn-common/types/cookie"
	"github.com/eduvpn/eduvpn-common/types/protocol"
)

// Type gives the type of server
type Type int8

const (
	// TypeUnknown means the server is unknown
	TypeUnknown Type = iota
	// TypeInstituteAccess means the server is of type Institute Access
	TypeInstituteAccess
	// TypeSecureInternet means the server is of type Secure Internet
	TypeSecureInternet
	// TypeCustom means the server is of type Custom Server
	TypeCustom
)

// This is here to support V1 configs which had the server type as a string
func (t *Type) UnmarshalJSON(data []byte) error {
	// First try to just unmarshal the type
	var num int8
	if err := json.Unmarshal(data, &num); err == nil {
		if num < int8(TypeUnknown) || num > int8(TypeCustom) {
			return fmt.Errorf("invalid server type: %d", num)
		}
		*t = Type(num)
		return nil
	}

	// unmarshal the old way, as a string
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "secure_internet":
		*t = TypeSecureInternet
	case "institute_access":
		*t = TypeInstituteAccess
	case "custom_server":
		*t = TypeCustom
	default:
		return fmt.Errorf("invalid server type: %s", str)
	}
	return nil
}

// RequiredAskTransition represents the data that is sent when a transition is required to be handled and the go library needs data from the client
// This data can be a profile selection or secure internet location selection
type RequiredAskTransition struct {
	// C is the cookie which is needed for replying to the library, see CookieReply in exports/exports.go
	// If this cookie is omitted, it is a protocol error
	C *cookie.Cookie `json:"cookie,omitempty"`
	// Data is the data associated to the transition, e.g. the list of profiles (Profiles struct)
	// or the list of secure internet locations ([]string)
	Data interface{} `json:"data"`
}

// Expiry is the struct that gives the time at which certain expiry elements should be shown
type Expiry struct {
	// StartTime is the start time of the VPN in Unix
	StartTime int64 `json:"start_time"`
	// EndTime is the end time of the VPN in Unix.
	EndTime int64 `json:"end_time"`
	// ButtonTime is the Unix time at which to start showing the renew button in the UI
	ButtonTime int64 `json:"button_time"`
	// CountdownTime is the Unix time at which to start showing more detailed countdown timer.
	// E.g. first start with days (7 days left), and when the current time is after this time, show e.g. 9 minutes and 59 seconds
	CountdownTime int64 `json:"countdown_time"`
	// NotificationTimes is the slice/list of times at which to show a notification that the VPN is about to expire
	NotificationTimes []int64 `json:"notification_times"`
}

// Profile is the profile for the VPN, to show in the UI where the user can switch to it to get a different VPN configuration
type Profile struct {
	// DisplayName is the display name of the profile as a map
	// It is a map where country codes are mapped to names, this is to be consistent with the format of other display names
	// E.g. {"en": "Default Profile"}
	// If this is empty, the field is omitted from the JSON
	DisplayName map[string]string `json:"display_name,omitempty"`
}

// Profiles is the map of profiles with the current defined
type Profiles struct {
	// Map, the map of profiles from profile ID to the profile contents
	// If this is empty, the field is omitted from the JSON
	Map map[string]Profile `json:"map,omitempty"`
	// Current is the current profile ID that is defined
	Current string `json:"current"`
}

// Tokens are the OAuth tokens for the server
type Tokens struct {
	// Access is the access token
	Access string `json:"access_token"`
	// Refresh is the refresh token
	Refresh string `json:"refresh_token"`
	// Expires is the Unix timestamp when the token expires
	Expires int64 `json:"expires_at"`
}

// Server is the basic type for a server. This is the base for secure internet and institute access. Custom servers are equal to this type
type Server struct {
	// DisplayName is the map from language tags to display name. If this is empty, the field is omitted from the JSON
	DisplayName map[string]string `json:"display_name,omitempty"`
	// Identifier is the Base URL for Institute Access and Custom Server. For Secure Internet this is the organization ID
	// This identifier should be passed to the Go library for e.g. getting a config
	Identifier string `json:"identifier"`
	// Profiles is the profiles that this server has defined
	// It could be that this is empty if the library has not discovered the profiles just yet
	Profiles Profiles `json:"profiles"`
}

// Institute defines an institute access server
type Institute struct {
	// Server is the embedded server struct
	Server
	// SupportContacts are the list of support contacts
	SupportContacts []string `json:"support_contacts,omitempty"`
	// Delisted is a boolean that indicates whether or not this server is delisted from discovery
	// If it is, the UI should show a warning symbol or move the server to a new category, which is up to the client
	Delisted bool `json:"delisted"`
}

// SecureInternet is a secure internet server
type SecureInternet struct {
	// Server is the embedded server struct
	Server
	// CountryCode is the country code of the currently configured location, e.g. "nl"
	CountryCode string `json:"country_code"`
	// Locations is the list of available secure internet locations
	Locations []string `json:"locations,omitempty"`
	// SupportContacts are the list of support contacts
	SupportContacts []string `json:"support_contacts,omitempty"`
	// Delisted is a boolean that indicates whether or not this server is delisted from discovery
	// If it is, the UI should show a warning symbol or move the server to a new category, which is up to the client
	Delisted bool `json:"delisted"`
}

// List is the list of servers
type List struct {
	// Institutes is the list/slice of institute access servers. If none are defined, this is omitted in the JSON
	Institutes []Institute `json:"institute_access_servers,omitempty"`
	// Secure Internet is the secure internet server if any. If none is there, it is omitted in the JSON
	SecureInternet *SecureInternet `json:"secure_internet_server,omitempty"`
	// Custom is the list/slice of custom servers. If none are defined, this is omitted in the JSON
	Custom []Server `json:"custom_servers,omitempty"`
}

// Configuration is the configuration that you get back when you call the get config function
type Configuration struct {
	// VPNConfig is the VPN Configuration, a WireGuard or OpenVPN Configuration
	// In case of OpenVPN, we append "script-security 0" to disable scripts from being run by default.
	// A client may override this, e.g. for, very trusted, pre-provisioned VPNs
	VPNConfig string `json:"config"`
	// Protocol defines which protocol the configuration is for, OpenVPN or WireGuard
	Protocol protocol.Protocol `json:"protocol"`
	// DefaultGateway is a boolean that indicates whether or not this configuration should be configured as a default gateway
	DefaultGateway bool `json:"default_gateway"`
	// DNSSearchDomains are the list of dns search domains
	DNSSearchDomains []string `json:"dns_search_domains,omitempty"`
}

// Current is the struct that defines the current server
// It has different fields where only two are always filled in
type Current struct {
	// The following three are mutually exclusive

	// Institute is the institute access server if any, if none is there this field is omitted in the JSON
	Institute *Institute `json:"institute_access_server,omitempty"`
	// Secure Internet is the secure internet server if any, if none is there this field is omitted in the JSON
	SecureInternet *SecureInternet `json:"secure_internet_server,omitempty"`
	// Custom is the custom server if any, if none is there this field is omitted in the JSON
	Custom *Server `json:"custom_server,omitempty"`
	// Type is the type of server that is there to check which of the three types should be non-nil
	Type Type `json:"server_type"`
}
