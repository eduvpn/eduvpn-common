// Package v1 implements a minimum set of the v1 config to convert it to a v2 config
// In version 1 of the config we used the internal state as the config
// This was bad as now if we want to change some internal representation the config also changes
// This package can be removed when most people have migrated from v1 to v2
package v1

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api/profiles"
	"github.com/eduvpn/eduvpn-common/internal/discovery"
	"github.com/eduvpn/eduvpn-common/types/server"
)

type Profiles struct {
	profiles.Info
	Current string `json:"current_profile"`
}

type Base struct {
	BaseURL        string    `json:"base_url"`
	Profiles       Profiles  `json:"profiles"`
	StartTime      time.Time `json:"start_time"`
	StartTimeOAuth time.Time `json:"start_time_oauth"`
	ExpireTime     time.Time `json:"expire_time"`
}

type InstituteServer struct {
	Base     Base     `json:"base"`
	Profiles Profiles `json:"profiles"`
}

type InstituteServers struct {
	Map        map[string]InstituteServer `json:"map"`
	CurrentURL string                     `json:"current_url"`
}

type (
	CustomServer  = InstituteServer
	CustomServers = InstituteServers
)

type SecureInternetHome struct {
	BaseMap            map[string]*Base  `json:"base_map"`
	DisplayName        map[string]string `json:"display_name"`
	HomeOrganizationID string            `json:"home_organization_id"`
	CurrentLocation    string            `json:"current_location"`
}

type Servers struct {
	Custom             CustomServers      `json:"custom_servers"`
	Institute          InstituteServers   `json:"institute_servers"`
	SecureInternetHome SecureInternetHome `json:"secure_internet_home"`
	IsType             server.Type        `json:"is_secure_internet"`
}

type V1 struct {
	Discovery discovery.Discovery `json:"discovery"`
	Servers   Servers             `json:"servers"`
}
