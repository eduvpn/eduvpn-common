package server

type Profile struct {
	ID             string   `json:"profile_id"`
	DisplayName    string   `json:"display_name"`
	VPNProtoList   []string `json:"vpn_proto_list"`
	DefaultGateway bool     `json:"default_gateway"`
}

type ProfileListInfo struct {
	ProfileList []Profile `json:"profile_list"`
}

type ProfileInfo struct {
	Current string          `json:"current_profile"`
	Info    ProfileListInfo `json:"info"`
}

func (info ProfileInfo) GetCurrentProfileIndex() int {
	index := 0
	for _, profile := range info.Info.ProfileList {
		if profile.ID == info.Current {
			return index
		}
		index++
	}
	// Default is 'first' profile
	return 0
}

func (profile *Profile) supportsProtocol(protocol string) bool {
	for _, proto := range profile.VPNProtoList {
		if proto == protocol {
			return true
		}
	}
	return false
}

func (profile *Profile) supportsWireguard() bool {
	return profile.supportsProtocol("wireguard")
}

func (profile *Profile) supportsOpenVPN() bool {
	return profile.supportsProtocol("openvpn")
}
