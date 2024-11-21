// Package profiles defines a wrapper around the various profiles
// returned by the /info endpoint
package profiles

import (
	"codeberg.org/eduVPN/eduvpn-common/types/protocol"
	"codeberg.org/eduVPN/eduvpn-common/types/server"
)

// Profile is the information for a profile
type Profile struct {
	// ID is the identifier of the profile
	// Used to select a profile
	ID string `json:"profile_id"`
	// DisplayName defines the UI friendly name for the profile
	DisplayName string `json:"display_name"`
	// VPNProtoList defines the list of VPN protocols
	// E.g. wireguard, openvpn
	VPNProtoList []string `json:"vpn_proto_list"`
	// VPNProtoTransportList defines the list of VPN protocols including their transport values
	// E.g. wireguard+udp, openvpn+tcp
	VPNProtoTransportList []string `json:"vpn_proto_transport_list"`
	// DefaultGateway specifies whether or not this profile is a default gateway profile
	DefaultGateway bool `json:"default_gateway"`
	// DNSSearchDomains specifies the list of dns search domains
	// This is provided for a Linux client issue
	// See: https://github.com/eduvpn/python-eduvpn-client/issues/550
	DNSSearchDomains []string `json:"dns_search_domain_list"`
}

// ListInfo is the struct that has the profile list
type ListInfo struct {
	ProfileList []Profile `json:"profile_list"`
}

// Info is the top-level struct for the info endpoint
type Info struct {
	Info ListInfo `json:"info"`
}

// Len returns the length of the profile list
func (i Info) Len() int {
	return len(i.Info.ProfileList)
}

// Get returns a profile with id `id`, it returns nil if it is not found
func (i Info) Get(id string) *Profile {
	for _, p := range i.Info.ProfileList {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

// MustIndex gets a profile by index
// This index must be in the bounds
func (i Info) MustIndex(n int) Profile {
	return i.Info.ProfileList[n]
}

func hasProtocol(protos []string, proto protocol.Protocol) bool {
	for _, p := range protos {
		if protocol.New(p) == proto {
			return true
		}
	}
	return false
}

// ShouldFailover returns whether or not this VPN profile should start a failover procedure
// This is true when the profile supports a TCP connection
// If we cannot determine whether it supports a TCP connection
// (because the server doesn't provide the VPN transport list function yet),
// we will just check if it supports OpenVPN
func (p *Profile) ShouldFailover() bool {
	// old servers don't support it, only failover in case OpenVPN is supported
	if len(p.VPNProtoTransportList) == 0 {
		// this checks VPNProtoList
		return p.HasOpenVPN()
	}
	for _, c := range p.VPNProtoTransportList {
		if c == "wireguard+tcp" {
			return true
		}
		if c == "openvpn+tcp" {
			return true
		}
	}
	return false
}

// HasOpenVPN returns whether or not the profile has OpenVPN support
func (p *Profile) HasOpenVPN() bool {
	return hasProtocol(p.VPNProtoList, protocol.OpenVPN)
}

// HasWireGuard returns whether or not the profile has WireGuard support
func (p *Profile) HasWireGuard() bool {
	return hasProtocol(p.VPNProtoList, protocol.WireGuard)
}

// Public gets the server list as a structure that we return to clients
func (i Info) Public() server.Profiles {
	m := make(map[string]server.Profile)
	for _, p := range i.Info.ProfileList {
		m[p.ID] = server.Profile{
			DisplayName: map[string]string{
				"en": p.DisplayName,
			},
			DefaultGateway: p.DefaultGateway,
		}
	}
	return server.Profiles{Map: m}
}
