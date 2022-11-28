package server

import (
	"time"

	"github.com/eduvpn/eduvpn-common/types"
)

// The base type for servers.
type Base struct {
	URL            string            `json:"base_url"`
	DisplayName    map[string]string `json:"display_name"`
	SupportContact []string          `json:"support_contact"`
	Endpoints      Endpoints         `json:"endpoints"`
	Profiles       ProfileInfo       `json:"profiles"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"expire_time"`
	Type           string            `json:"server_type"`
}

func (base *Base) InitializeEndpoints() error {
	errorMessage := "failed initializing endpoints"
	endpoints, endpointsErr := APIGetEndpoints(base.URL)
	if endpointsErr != nil {
		return types.NewWrappedError(errorMessage, endpointsErr)
	}
	base.Endpoints = *endpoints
	return nil
}

func (base *Base) ValidProfiles(clientSupportsWireguard bool) ProfileInfo {
	var validProfiles []Profile
	for _, profile := range base.Profiles.Info.ProfileList {
		// Not a valid profile because it does not support openvpn
		// Also the client does not support wireguard
		if !profile.supportsOpenVPN() && !clientSupportsWireguard {
			continue
		}
		validProfiles = append(validProfiles, profile)
	}
	return ProfileInfo{
		Current: base.Profiles.Current,
		Info:    ProfileListInfo{ProfileList: validProfiles},
	}
}
