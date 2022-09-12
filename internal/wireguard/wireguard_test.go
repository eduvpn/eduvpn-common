package wireguard

import (
	"fmt"
	"testing"
)

func Test_ConfigAddKey(t *testing.T) {
	config := `
[Interface]

[interface]

[interface2]

interface

 [Interface]

[Interface]test
`
	wgKey, wgKeyErr := GenerateKey()

	if wgKeyErr != nil {
		t.Fatalf("WireGuard config add key, generate key error: %v", wgKeyErr)
	}
	expectedConfig := fmt.Sprintf(`
[Interface]
PrivateKey = %s

[interface]

[interface2]

interface

 [Interface]

[Interface]test
`, wgKey.String())
	gotConfig := ConfigAddKey(config, wgKey)

	if gotConfig != expectedConfig {
		t.Fatalf("Got: %s, Want: %s", gotConfig, expectedConfig)
	}
}
