package eduvpn

import (
	"fmt"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"regexp"
)

func WireguardGenerateKey() (wgtypes.Key, error) {
	key, error := wgtypes.GeneratePrivateKey()
	return key, error
}

// FIXME: Instead of doing a regex replace, decide if we should use a parser
func WireguardConfigAddKey(config string, key wgtypes.Key) string {
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
