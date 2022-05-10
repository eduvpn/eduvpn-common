package internal

import (
	"fmt"
	"regexp"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func wireguardGenerateKey() (wgtypes.Key, error) {
	key, keyErr := wgtypes.GeneratePrivateKey()

	if keyErr != nil {
		return key, &WireguardGenerateKeyError{Err: keyErr}
	}
	return key, nil
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

func WireguardGetConfig(server Server, supportsOpenVPN bool) (string, error) {
	base, baseErr := server.GetBase()

	if baseErr != nil {
		return "", &WireguardGetConfigError{Err: baseErr}
	}

	profile_id := base.Profiles.Current
	wireguardKey, wireguardErr := wireguardGenerateKey()

	if wireguardErr != nil {
		return "", &WireguardGetConfigError{Err: wireguardErr}
	}

	wireguardPublicKey := wireguardKey.PublicKey().String()
	config, content, _, configErr := APIConnectWireguard(server, profile_id, wireguardPublicKey, supportsOpenVPN)

	if configErr != nil {
		return "", &WireguardGetConfigError{Err: wireguardErr}
	}

	if content == "wireguard" {
		// FIXME: Store expiry
		// This needs the go code a way to identify a connection
		// Use the uuid of the connection e.g. on Linux
		// This needs the client code to call the go code

		config = wireguardConfigAddKey(config, wireguardKey)
	}

	return config, nil
}

type WireguardGenerateKeyError struct {
	Err error
}

func (e *WireguardGenerateKeyError) Error() string {
	return fmt.Sprintf("failed generating Wireguard key with error: %v", e.Err)
}

type WireguardGetConfigError struct {
	Err error
}

func (e *WireguardGetConfigError) Error() string {
	return fmt.Sprintf("failed getting Wireguard config with error: %v", e.Err)
}
