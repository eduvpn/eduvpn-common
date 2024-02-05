package client

// isAllowedClientID checks if the 'clientID' is in the list of allowed client IDs
func isAllowedClientID(clientID string) bool {
	allowList := []string{
		// eduVPN
		"org.eduvpn.app.windows",
		"org.eduvpn.app.android",
		"org.eduvpn.app.ios",
		"org.eduvpn.app.macos",
		"org.eduvpn.app.linux",
		// Let's Connect!
		"org.letsconnect-vpn.app.windows",
		"org.letsconnect-vpn.app.android",
		"org.letsconnect-vpn.app.ios",
		"org.letsconnect-vpn.app.macos",
		"org.letsconnect-vpn.app.linux",
		// govVPN
		"org.govvpn.app.windows",
		"org.govvpn.app.android",
		"org.govvpn.app.ios",
		"org.govvpn.app.macos",
		"org.govvpn.app.linux",
	}
	for _, x := range allowList {
		if x == clientID {
			return true
		}
	}
	return false
}

func userAgentName(clientID string) string {
	switch clientID {
	case "org.eduvpn.app.windows":
		return "eduVPN for Windows"
	case "org.eduvpn.app.android":
		return "eduVPN for Android"
	case "org.eduvpn.app.ios":
		return "eduVPN for iOS"
	case "org.eduvpn.app.macos":
		return "eduVPN for macOS"
	case "org.eduvpn.app.linux":
		return "eduVPN for Linux"
	case "org.letsconnect-vpn.app.windows":
		return "Let's Connect! for Windows"
	case "org.letsconnect-vpn.app.android":
		return "Let's Connect! for Android"
	case "org.letsconnect-vpn.app.ios":
		return "Let's Connect! for iOS"
	case "org.letsconnect-vpn.app.macos":
		return "Let's Connect! for macOS"
	case "org.letsconnect-vpn.app.linux":
		return "Let's Connect! for Linux"
	case "org.govvpn.app.windows":
		return "govVPN for Windows"
	case "org.govvpn.app.android":
		return "govVPN for Android"
	case "org.govvpn.app.ios":
		return "govVPN for iOS"
	case "org.govvpn.app.macos":
		return "govVPN for macOS"
	case "org.govvpn.app.linux":
		return "govVPN for Linux"
	default:
		return "unknown"
	}
}
