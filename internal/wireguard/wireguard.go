package wireguard

import (
	"fmt"
	"regexp"

	"github.com/eduvpn/eduvpn-common/types"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func GenerateKey() (wgtypes.Key, error) {
	key, keyErr := wgtypes.GeneratePrivateKey()

	if keyErr != nil {
		return key, types.NewWrappedError(
			"failed generating WireGuard key",
			keyErr,
		)
	}
	return key, nil
}

// FIXME: Instead of doing a regex replace, decide if we should use a parser
func ConfigAddKey(config string, key wgtypes.Key) string {
	interfaceSection := "[Interface]"
	InterfaceSectionEscaped := regexp.QuoteMeta(interfaceSection)

	// (?m) enables multi line mode
	// ^ match from beginning of line
	// $ match till end of line
	// So it matches [Interface] section exactly
	InterfaceRe := regexp.MustCompile(fmt.Sprintf("(?m)^%s$", InterfaceSectionEscaped))
	toReplace := fmt.Sprintf("%s\nPrivateKey = %s", interfaceSection, key.String())
	return InterfaceRe.ReplaceAllString(config, toReplace)
}
