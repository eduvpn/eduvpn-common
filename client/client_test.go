package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/types/cookie"
	"github.com/eduvpn/eduvpn-common/types/protocol"
	srvtypes "github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

func getServerURI(t *testing.T) string {
	serverURI := os.Getenv("SERVER_URI")
	if serverURI == "" {
		t.Skip("Skipping server test as no SERVER_URI env var has been passed")
	}
	serverURI, parseErr := httpw.EnsureValidURL(serverURI, true)
	if parseErr != nil {
		t.Skip("Skipping server test as the server uri is not valid")
	}
	return serverURI
}

func runCommand(errBuffer *strings.Builder, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Stderr = errBuffer
	err := cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func loginOAuthSelenium(ck *cookie.Cookie, url string) {
	// We could use the go selenium library
	// But it does not support the latest selenium v4 just yet
	var errBuffer strings.Builder
	err := runCommand(&errBuffer, "python3", "../selenium_eduvpn.py", url)
	if err != nil {
		_ = ck.Cancel()
		panic(fmt.Sprintf(
			"Login OAuth with selenium script failed with error %v and stderr %s",
			err,
			errBuffer.String(),
		))
	}
}

func stateCallback(
	ck *cookie.Cookie,
	_ FSMStateID,
	newState FSMStateID,
	data interface{},
) {
	if newState == StateOAuthStarted {
		url, ok := data.(string)

		if !ok {
			panic("data is not a string for OAuth URL")
		}
		loginOAuthSelenium(ck, url)
	}
}

func TestServer(t *testing.T) {
	serverURI := getServerURI(t)
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	dir := t.TempDir()
	var state *Client
	state, err := New(
		"org.letsconnect-vpn.app.linux",
		"0.1.0-test",
		dir,
		func(oldState FSMStateID, newState FSMStateID, data interface{}) bool {
			// test if main server server list succeeds
			if newState == StateMain {
				_, listErr := state.ServerList()
				if listErr != nil {
					t.Fatalf("Got server list error: %v", listErr)
				}
			}
			go stateCallback(ck, oldState, newState, data)
			return true
		},
		false,
	)
	if err != nil {
		t.Fatalf("Creating client error: %v", err)
	}
	err = state.Register()
	if err != nil {
		t.Fatalf("Registering error: %v", err)
	}

	addErr := state.AddServer(ck, serverURI, srvtypes.TypeCustom, nil)
	if addErr != nil {
		t.Fatalf("Add error: %v", addErr)
	}
	_, configErr := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)
	if configErr != nil {
		t.Fatalf("Connect error: %v", configErr)
	}
}

func TestTokenExpired(t *testing.T) {
	serverURI := getServerURI(t)
	expiredTTL := os.Getenv("OAUTH_EXPIRED_TTL")
	if expiredTTL == "" {
		t.Log(
			"No expired TTL present, skipping this test. Set OAUTH_EXPIRED_TTL env variable to run this test",
		)
		return
	}

	// Convert the env variable to an int and signal error if it is not possible
	expiredInt, expiredErr := strconv.Atoi(expiredTTL)
	if expiredErr != nil {
		t.Fatalf("Cannot convert EXPIRED_TTL env variable to an int with error %v", expiredErr)
	}

	// Get a vpn state
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	dir := t.TempDir()
	state, err := New(
		"org.letsconnect-vpn.app.linux",
		"0.1.0-test",
		dir,
		func(oldState FSMStateID, newState FSMStateID, data interface{}) bool {
			go stateCallback(ck, oldState, newState, data)
			return true
		},
		false,
	)
	if err != nil {
		t.Fatalf("Creating client error: %v", err)
	}
	err = state.Register()
	if err != nil {
		t.Fatalf("Registering error: %v", err)
	}

	addErr := state.AddServer(ck, serverURI, srvtypes.TypeCustom, nil)
	if addErr != nil {
		t.Fatalf("Add error: %v", addErr)
	}

	_, configErr := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)

	if configErr != nil {
		t.Fatalf("Connect error before expired: %v", configErr)
	}

	// get token before
	tb, err := state.retrieveTokens(serverURI, srvtypes.TypeCustom)
	if err != nil {
		t.Fatalf("No tokens found: %v", err)
	}

	// Wait for TTL so that the tokens expire
	time.Sleep(time.Duration(expiredInt) * time.Second)

	_, configErr = state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)

	if configErr != nil {
		t.Fatalf("Connect error after expiry: %v", configErr)
	}

	ta, err := state.retrieveTokens(serverURI, srvtypes.TypeCustom)
	if err != nil {
		t.Fatalf("No tokens found after: %v", err)
	}
	if tb.Access == ta.Access {
		t.Errorf("Access token is the same after refresh")
	}
}

// Test if an invalid profile will be corrected.
func TestInvalidProfileCorrected(t *testing.T) {
	dir := t.TempDir()
	serverURI := getServerURI(t)
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	state, err := New(
		"org.letsconnect-vpn.app.linux",
		"0.1.0-test",
		dir,
		func(oldState FSMStateID, newState FSMStateID, data interface{}) bool {
			go stateCallback(ck, oldState, newState, data)
			return true
		},
		false,
	)
	if err != nil {
		t.Fatalf("Creating client error: %v", err)
	}
	err = state.Register()
	if err != nil {
		t.Fatalf("Registering error: %v", err)
	}

	addErr := state.AddServer(ck, serverURI, srvtypes.TypeCustom, nil)
	if addErr != nil {
		t.Fatalf("Add error: %v", addErr)
	}

	_, configErr := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)

	if configErr != nil {
		t.Fatalf("First connect error: %v", configErr)
	}

	s, err := state.Servers.CurrentServer()
	if err != nil {
		t.Fatalf("Got error when getting current server: %v", err)
	}
	// set invalid profile
	invp := "IDONOTEXIST"
	s.Profiles.Current = invp

	_, configErr = state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)
	if configErr != nil {
		t.Fatalf("Second connect error: %v", configErr)
	}

	if s.Profiles.Current == invp {
		t.Fatalf(
			"Profiles do no match: current %s and previous %s",
			s.Profiles.Current,
			invp,
		)
	}
}

// TestConfigStartup tests if the 'startup' variable for getconfig behaves as expected
func TestConfigStartup(t *testing.T) {
	serverURI := getServerURI(t)
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	dir := t.TempDir()
	state, err := New(
		"org.letsconnect-vpn.app.linux",
		"0.1.0-test",
		dir,
		func(oldState FSMStateID, newState FSMStateID, data interface{}) bool {
			go stateCallback(ck, oldState, newState, data)
			return true
		},
		false,
	)
	if err != nil {
		t.Fatalf("Creating client error: %v", err)
	}
	err = state.Register()
	if err != nil {
		t.Fatalf("Failed to register with error: %v", err)
	}
	// we set true as last argument here such that no callbacks are ran
	var ot int64 = 5
	err = state.AddServer(ck, serverURI, srvtypes.TypeCustom, &ot)
	if err != nil {
		t.Fatalf("Failed to add server for trying config startup: %v", err)
	}
	testTrue := func() {
		// Now get config with setting startup to true
		startup := true
		_, err := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, startup)
		// this should fail as we have not authorized yet/chosen profile and startup=true does not do these callbacks
		if err == nil {
			t.Fatal("Got no error after getting config with startup true")
		}
		if !strings.HasPrefix(err.Error(), "The client tried to autoconnect to the VPN server") {
			t.Fatalf("GetConfig error for GetConfig with startup=true is not what we expect: %v", err)
		}
	}
	testFalse := func() {
		startup := false
		_, err := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, startup)
		if err != nil {
			t.Fatalf("Got error after getting config with startup=false: %v", err)
		}
	}
	testTrue()
	testFalse()

	// set invalid authorization and test again
	// we cannot test by setting invalid profile because the server only has 1 profile
	err = state.tokCacher.Set(serverURI, srvtypes.TypeCustom, eduoauth.Token{})
	if err != nil {
		t.Fatalf("Failed to set token cache: %v", err)
	}
	testTrue()
	testFalse()
}

// Test if prefer tcp is handled correctly by checking the returned config and config type.
func TestPreferTCP(t *testing.T) {
	serverURI := getServerURI(t)
	ck := cookie.NewWithContext(context.Background())
	defer ck.Cancel() //nolint:errcheck
	dir := t.TempDir()
	state, err := New(
		"org.letsconnect-vpn.app.linux",
		"0.1.0-test",
		dir,
		func(oldState FSMStateID, newState FSMStateID, data interface{}) bool {
			go stateCallback(ck, oldState, newState, data)
			return true
		},
		false,
	)
	if err != nil {
		t.Fatalf("Creating client error: %v", err)
	}
	err = state.Register()
	if err != nil {
		t.Fatalf("Registering error: %v", err)
	}

	addErr := state.AddServer(ck, serverURI, srvtypes.TypeCustom, nil)
	if addErr != nil {
		t.Fatalf("Add error: %v", addErr)
	}

	// get a config with preferTCP set to true
	config, configErr := state.GetConfig(ck, serverURI, srvtypes.TypeCustom, true, false)

	// Test server should accept prefer TCP!
	if config.Protocol != protocol.OpenVPN && config.Protocol != protocol.WireGuardProxy {
		t.Fatalf("Invalid protocol for prefer TCP")
	}

	if configErr != nil {
		t.Fatalf("Config error: %v", configErr)
	}

	// We also test for script security 0 here
	if config.Protocol == protocol.OpenVPN && !strings.HasSuffix(config.VPNConfig, "udp\nscript-security 0") {
		t.Fatalf("Suffix for prefer TCP is not in the right order for config: %s", config.VPNConfig)
	}

	// get a config with preferTCP set to false
	config, configErr = state.GetConfig(ck, serverURI, srvtypes.TypeCustom, false, false)
	if configErr != nil {
		t.Fatalf("Config error: %v", configErr)
	}

	// We also test for script security 0 here
	if config.Protocol == protocol.OpenVPN &&
		!strings.HasSuffix(config.VPNConfig, "tcp\nscript-security 0") {
		t.Fatalf("Suffix for disable prefer TCP is not in the right order for config: %s", config.VPNConfig)
	}
}

func TestInvalidClientID(t *testing.T) {
	tests := map[string]bool{
		"test":                          false,
		"org.letsconnect-vpn.app.linux": true,
		"org.letsconnect-vpn":           false,
		"org.letsconnect-vpn.app":       false,
		"org.letsconnect-vpn.linuxsd":   false,
		"org.letsconnect-vpn.app.macos": true,
	}

	dir := t.TempDir()
	for k, v := range tests {
		_, err := New(
			k,
			"0.1.0-test",
			dir,
			func(_ FSMStateID, _ FSMStateID, _ interface{}) bool {
				return true
			},
			false,
		)
		if v {
			if err != nil {
				t.Fatalf("expected valid register with clientID: %v, got error: %v", k, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("expected invalid register with clientID: %v, but got no error", k)
		}
		if !strings.HasPrefix(err.Error(), "An internal error occurred. The cause of the error is: The client registered with an invalid client ID") {
			t.Fatalf("register error has invalid prefix: %v", err.Error())
		}
	}
}
