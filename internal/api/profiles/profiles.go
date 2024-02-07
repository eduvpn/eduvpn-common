package profiles

import (
	"github.com/eduvpn/eduvpn-common/types/protocol"
	"github.com/eduvpn/eduvpn-common/types/server"
)

type Profile struct {
	ID                    string   `json:"profile_id"`
	DisplayName           string   `json:"display_name"`
	VPNProtoList          []string `json:"vpn_proto_list"`
	VPNProtoTransportList []string `json:"vpn_proto_transport_list"`
	DefaultGateway        bool     `json:"default_gateway"`
	DNSSearchDomains      []string `json:"dns_search_domain_list"`
}

type ListInfo struct {
	ProfileList []Profile `json:"profile_list"`
}

type Info struct {
	Info ListInfo `json:"info"`
}

func (i Info) Len() int {
	return len(i.Info.ProfileList)
}

func (i Info) Get(id string) *Profile {
	for _, p := range i.Info.ProfileList {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

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

func (p *Profile) HasOpenVPN() bool {
	return hasProtocol(p.VPNProtoList, protocol.OpenVPN)
}

func (p *Profile) HasWireGuard() bool {
	return hasProtocol(p.VPNProtoList, protocol.WireGuard)
}

func (i Info) FilterWireGuard() *Info {
	var ret []Profile
	for _, p := range i.Info.ProfileList {
		if !p.HasOpenVPN() {
			continue
		}
	}
	return &Info{
		Info: ListInfo{
			ProfileList: ret,
		},
	}
}

func (i Info) Public() server.Profiles {
	m := make(map[string]server.Profile)
	for _, p := range i.Info.ProfileList {
		m[p.ID] = server.Profile{
			DisplayName: map[string]string{
				"en": p.DisplayName,
			},
		}
	}
	return server.Profiles{Map: m}
}
