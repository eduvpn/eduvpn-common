package eduvpn

import (
	"fmt"
	"regexp"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func wireguardGenerateKey() (wgtypes.Key, error) {
	key, error := wgtypes.GeneratePrivateKey()
	return key, error
}

// FIXME: Instead of doing a regex replace, decide if we should use a parser
func wireguardConfigAddKey(config string, key wgtypes.Key) string {
	interface_section := "[Interface]"
	interface_section_escaped := regexp.QuoteMeta(interface_section)

	// (?m) enables multi line mode
	// ^ match from beginning of line
	// $ match till end of line
	// So it matches [Interface] section exactly
	interface_re := regexp.MustCompile(fmt.Sprintf("(?m)^%s$", interface_section_escaped))
	to_replace := fmt.Sprintf("%s\nPrivateKey = %s", interface_section, key.String())
	return interface_re.ReplaceAllString(config, to_replace)
}

func (server *Server) WireguardGetConfig() (string, error) {
	wireguardKey, wireguardErr := wireguardGenerateKey()

	if wireguardErr != nil {
		return "", wireguardErr
	}

	wireguardPublicKey := wireguardKey.PublicKey().String()
	configWireguard, _, configErr := server.APIConnectWireguard("default", wireguardPublicKey)

	if configErr != nil {
		return "", configErr
	}

	// FIXME: Store expiry
	// This needs the go code a way to identify a connection
	// Use the uuid of the connection e.g. on Linux
	// This needs the client code to call the go code

	configWireguardKey := wireguardConfigAddKey(configWireguard, wireguardKey)

	return configWireguardKey, nil
}
