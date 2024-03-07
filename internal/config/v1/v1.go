// Package v1 implements a minimum set of the v1 config to convert it to a v2 config
// In version 1 of the config we used the internal state as the config
// This was bad as now if we want to change some internal representation the config also changes
// This package can be removed when most people have migrated from v1 to v2
package v1

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api/profiles"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
)

// Profiles is the list of profiles
type Profiles struct {
	profiles.Info
	Current string `json:"current_profile"`
}

// Base is the base type of a server
type Base struct {
	// BaseURL is the base_url from discovery
	BaseURL string `json:"base_url"`
	// profiles is the last of profile
	Profiles Profiles `json:"profiles"`
	// StartTime is the time when we started the connection
	StartTime time.Time `json:"start_time"`
	// StartTimeOAuth is the time when we last started OAuth
	StartTimeOAuth time.Time `json:"start_time_oauth"`
	// ExpireTime is the time when the connection expires
	ExpireTime time.Time `json:"expire_time"`
}

// InstituteServer is the struct that represents an institute access server
type InstituteServer struct {
	// Base is the base of the server
	Base Base `json:"base"`
}

// InstituteServers is a list of institute access servers
type InstituteServers struct {
	// Map is the map from base url to an institute access server
	Map map[string]InstituteServer `json:"map"`
	// CurrentURL is the URL of the currently configured server
	CurrentURL string `json:"current_url"`
}

type (
	// CustomServer is an alias to InstituteServer
	CustomServer = InstituteServer
	// CustomServers is an alias to InstituteServers
	CustomServers = InstituteServers
)

// SecureInternetHome represents a secure internet home server
type SecureInternetHome struct {
	// BaseMap is the map from country code to a server base
	BaseMap map[string]*Base `json:"base_map"`
	// DisplayName is the map from language code to UI name
	DisplayName map[string]string `json:"display_name"`
	// HomeOrganizationID is the identifier of the home organization
	HomeOrganizationID string `json:"home_organization_id"`
	// CurrentLocation is the country code of the currently configured server
	CurrentLocation string `json:"current_location"`
}

type Type int8

const (
	CustomServerType Type = iota
	InstituteAccessServerType
	SecureInternetServerType
)

// Servers represents the list of servers
type Servers struct {
	// Custom are the "custom" servers; the servers that are added by the user
	Custom CustomServers `json:"custom_servers"`
	// Institute are the institute access servers configured from discovery
	Institute InstituteServers `json:"institute_servers"`
	// SecureInternetHome is the secure internet home server
	// Also obtained through discovery
	SecureInternetHome SecureInternetHome `json:"secure_internet_home"`
	// IsType represents which server type was last configured
	IsType Type `json:"is_secure_internet"`
}

// V1 is the top-level struct for the first version of the state file
type V1 struct {
	// Discovery is the list of discovery servers
	Discovery discovery.Discovery `json:"discovery"`
	// Servers is the list of servers in the app
	Servers Servers `json:"servers"`
}
