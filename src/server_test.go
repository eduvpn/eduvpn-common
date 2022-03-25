package eduvpn

import (
	"errors"
	"fmt"
	"testing"
	"net/http"
	"crypto/tls"
	"os/exec"
	"strings"
)

func runCommand(t *testing.T, errBuffer *strings.Builder, name string, args ...string) error {
	cmd := exec.Command(name, args...)

	cmd.Stderr = errBuffer
	err := cmd.Start()
	if err != nil {
		return err
	}

	return cmd.Wait()
}

func LoginOAuthSelenium(t* testing.T, url string) {
	// We could use the go selenium library
	// But it does not support the latest selenium v4 just yet
	var errBuffer strings.Builder
	err := runCommand(t, &errBuffer, "python3", "../selenium_eduvpn.py", url)

	if err != nil {
		t.Errorf("Login OAuth with selenium script failed with error %v and stderr %s", err, errBuffer.String())
	}
}

func StateCallback(t *testing.T, oldState string, newState string, data string) {
	if newState == "OAuthInitialized" {
		LoginOAuthSelenium(t, data)
	}
}

func Test_server(t *testing.T) {
	state := GetVPNState()

	// Do not verify because during testing, the cert is self-signed
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	state.Register("org.eduvpn.app.linux", "configs", func(old string, new string, data string) {
		StateCallback(t, old, new, data)
	})
	_, configErr := state.Connect("https://eduvpnserver")

	if configErr != nil {
		t.Errorf("Connect error: %v", configErr)
	}
}

func test_connect_oauth_parameter(t* testing.T, parameters URLParameters, expectedErr interface{}) {
	state := &VPNState{}

	// Do not verify because during testing, the cert is self-signed
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	state.Register("org.eduvpn.app.linux", "configsnologin", func(old string, new string, data string) {
		if new == "OAuthInitialized" {
			baseURL := "http://127.0.0.1:8000/callback"
			url, err := HTTPConstructURL(baseURL, parameters)
			if err != nil {
				t.Errorf("Error: Constructing url %s with parameters %s", baseURL, fmt.Sprint(parameters))
			}
			_, _ = http.Get(url)
		}
	})
	_, configErr := state.Connect("https://eduvpnserver")

	if !errors.As(configErr, expectedErr) {
		t.Errorf("error %T = %v, wantErr %T", configErr, configErr, expectedErr)
	}
}

func Test_connect_oauth_parameters(t* testing.T) {

	var (
		failedCallbackParameterError *OAuthFailedCallbackParameterError
		failedCallbackStateMatchError *OAuthFailedCallbackStateMatchError
	)

	tests := []struct {
		expectedErr interface{}
		parameters URLParameters
	}{
		{&failedCallbackParameterError, URLParameters{}},
		{&failedCallbackParameterError, URLParameters{"code": "42"}},
		{&failedCallbackStateMatchError, URLParameters{"code": "42", "state": "21",}},
	}

	for _, test := range tests {
		test_connect_oauth_parameter(t, test.parameters, test.expectedErr)
	}
}