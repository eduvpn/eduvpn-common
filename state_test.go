package eduvpn

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jwijenbergh/eduvpn-common/internal"
)

func ensureLocalWellKnown() {
	wellKnown := os.Getenv("SERVER_IS_LOCAL")

	if wellKnown == "1" {
		internal.WellKnownPath = "well-known.php"
	}
}

func getServerURI(t *testing.T) string {
	serverURI := os.Getenv("SERVER_URI")
	if serverURI == "" {
		t.Skip("Skipping server test as no SERVER_URI env var has been passed")
	}
	return serverURI
}

func runCommand(t *testing.T, errBuffer *strings.Builder, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Stderr = errBuffer
	err := cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func loginOAuthSelenium(t *testing.T, url string, state *VPNState) {
	// We could use the go selenium library
	// But it does not support the latest selenium v4 just yet
	var errBuffer strings.Builder
	err := runCommand(t, &errBuffer, "python3", "selenium_eduvpn.py", url)
	if err != nil {
		t.Fatalf("Login OAuth with selenium script failed with error %v and stderr %s", err, errBuffer.String())
		state.CancelOAuth()
	}
}

func stateCallback(t *testing.T, oldState string, newState string, data string, state *VPNState) {
	if newState == "OAuth_Started" {
		loginOAuthSelenium(t, data, state)
	}
}

func Test_server(t *testing.T) {
	serverURI := getServerURI(t)
	state := &VPNState{}
	ensureLocalWellKnown()

	state.Register("org.eduvpn.app.linux", "configstest", func(old string, new string, data string) {
		stateCallback(t, old, new, data, state)
	}, false)

	_, configErr := state.ConnectInstituteAccess(serverURI)

	if configErr != nil {
		t.Fatalf("Connect error: %v", configErr)
	}
}

func test_connect_oauth_parameter(t *testing.T, parameters internal.URLParameters, expectedErr interface{}) {
	serverURI := getServerURI(t)
	state := &VPNState{}
	configDirectory := "test_oauth_parameters"

	state.Register("org.eduvpn.app.linux", configDirectory, func(oldState string, newState string, data string) {
		if newState == "OAuth_Started" {
			baseURL := "http://127.0.0.1:8000/callback"
			url, err := internal.HTTPConstructURL(baseURL, parameters)
			if err != nil {
				t.Fatalf("Error: Constructing url %s with parameters %s", baseURL, fmt.Sprint(parameters))
			}
			go http.Get(url)

		}
	}, false)
	_, configErr := state.ConnectInstituteAccess(serverURI)

	var stateErr *StateConnectError
	var loginErr *internal.OAuthLoginError
	var finishErr *internal.OAuthFinishError

	// We go through the chain of errors by unwrapping them one by one

	// First ensure we get a state connect error
	if !errors.As(configErr, &stateErr) {
		t.Fatalf("error %T = %v, wantErr %T", configErr, configErr, stateErr)
	}

	// Then ensure we get a login error
	gotLoginErr := stateErr.Err

	if !errors.As(gotLoginErr, &loginErr) {
		t.Fatalf("error %T = %v, wantErr %T", gotLoginErr, gotLoginErr, loginErr)
	}

	// Then ensure we get a finish error
	gotFinishErr := loginErr.Err

	if !errors.As(gotFinishErr, &finishErr) {
		t.Fatalf("error %T = %v, wantErr %T", gotFinishErr, gotFinishErr, finishErr)
	}

	// Then ensure we get the expected inner error
	gotExpectedErr := finishErr.Err

	if !errors.As(gotExpectedErr, expectedErr) {
		t.Fatalf("error %T = %v, wantErr %T", gotExpectedErr, gotExpectedErr, expectedErr)
	}
}

func Test_connect_oauth_parameters(t *testing.T) {
	var (
		failedCallbackParameterError  *internal.OAuthCallbackParameterError
		failedCallbackStateMatchError *internal.OAuthCallbackStateMatchError
	)

	tests := []struct {
		expectedErr interface{}
		parameters  internal.URLParameters
	}{
		{&failedCallbackParameterError, internal.URLParameters{}},
		{&failedCallbackParameterError, internal.URLParameters{"code": "42"}},
		{&failedCallbackStateMatchError, internal.URLParameters{"code": "42", "state": "21"}},
	}

	ensureLocalWellKnown()

	for _, test := range tests {
		test_connect_oauth_parameter(t, test.parameters, test.expectedErr)
	}
}

func Test_token_expired(t *testing.T) {
	serverURI := getServerURI(t)
	expiredTTL := os.Getenv("OAUTH_EXPIRED_TTL")
	if expiredTTL == "" {
		t.Log("No expired TTL present, skipping this test. Set EXPIRED_TTL env variable to run it")
		return
	}

	ensureLocalWellKnown()

	// Convert the env variable to an int and signal error if it is not possible
	expiredInt, expiredErr := strconv.Atoi(expiredTTL)
	if expiredErr != nil {
		t.Fatalf("Cannot convert EXPIRED_TTL env variable to an int with error %v", expiredErr)
	}

	// Get a vpn state
	state := &VPNState{}

	state.Register("org.eduvpn.app.linux", "configsexpired", func(old string, new string, data string) {
		stateCallback(t, old, new, data, state)
	}, false)

	_, configErr := state.ConnectInstituteAccess(serverURI)

	if configErr != nil {
		t.Fatalf("Connect error before expired: %v", configErr)
	}

	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		t.Fatalf("No server found")
	}

	oauth := server.GetOAuth()

	accessToken := oauth.Token.Access
	refreshToken := oauth.Token.Refresh

	// Wait for TTL so that the tokens expire
	time.Sleep(time.Duration(expiredInt) * time.Second)

	infoErr := internal.APIInfo(server)

	if infoErr != nil {
		t.Fatalf("Info error after expired: %v", infoErr)
	}

	// Check if tokens have changed
	accessTokenAfter := oauth.Token.Access
	refreshTokenAfter := oauth.Token.Refresh

	if accessToken == accessTokenAfter {
		t.Errorf("Access token is the same after refresh")
	}

	if refreshToken == refreshTokenAfter {
		t.Errorf("Refresh token is the same after refresh")
	}
}

func Test_token_invalid(t *testing.T) {
	serverURI := getServerURI(t)
	state := &VPNState{}

	ensureLocalWellKnown()

	state.Register("org.eduvpn.app.linux", "configsinvalid", func(old string, new string, data string) {
		stateCallback(t, old, new, data, state)
	}, false)

	_, configErr := state.ConnectInstituteAccess(serverURI)

	if configErr != nil {
		t.Fatalf("Connect error before invalid: %v", configErr)
	}

	// Fake connect and then back to authorized so that we can re-authorize
	// Going to authorized fakes a disconnect
	state.FSM.GoTransition(internal.CONNECTED)
	state.FSM.GoTransition(internal.AUTHORIZED)

	dummy_value := "37"

	server, serverErr := state.Servers.GetCurrentServer()
	if serverErr != nil {
		t.Fatalf("No server found")
	}

	oauth := server.GetOAuth()

	// Override tokens with invalid values
	oauth.Token.Access = dummy_value
	oauth.Token.Refresh = dummy_value

	infoErr := internal.APIInfo(server)

	if infoErr != nil {
		t.Fatalf("Info error after invalid: %v", infoErr)
	}

	if oauth.Token.Access == dummy_value {
		t.Errorf("Access token is equal to dummy value: %s", dummy_value)
	}

	if oauth.Token.Refresh == dummy_value {
		t.Errorf("Refresh token is equal to dummy value: %s", dummy_value)
	}
}
