package server

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/http"
)

// Base is the base type for servers.
type Base struct {
	URL            string            `json:"base_url"`
	DisplayName    map[string]string `json:"display_name"`
	SupportContact []string          `json:"support_contact"`
	Endpoints      Endpoints         `json:"endpoints"`
	Profiles       ProfileInfo       `json:"profiles"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        time.Time         `json:"expire_time"`
	Type           string            `json:"server_type"`
	httpClient     *http.Client
}

func (b *Base) InitializeEndpoints() error {
	ep, err := APIGetEndpoints(b.URL, b.httpClient)
	if err != nil {
		return err
	}
	b.Endpoints = *ep
	return nil
}

func (b *Base) ValidProfiles(wireguardSupport bool) ProfileInfo {
	var valid []Profile
	for _, p := range b.Profiles.Info.ProfileList {
		// Not a valid profile because it does not support openvpn
		// Also the client does not support wireguard
		if !p.SupportsOpenVPN() && !wireguardSupport {
			continue
		}
		valid = append(valid, p)
	}
	return ProfileInfo{
		Current: b.Profiles.Current,
		Info:    ProfileListInfo{ProfileList: valid},
	}
}
