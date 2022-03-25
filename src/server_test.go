package eduvpn

import (
	"testing"
	"net/http"
	"crypto/tls"
	"os/exec"
	"strings"
)

func RunCommand(t *testing.T, name string, args ...string) {
	cmd := exec.Command(name, args...)
	var errBuffer strings.Builder

	cmd.Stderr = &errBuffer
	err := cmd.Start()
	if err != nil {
		t.Errorf("%v", err)
	}

	err = cmd.Wait()

	if err != nil {
		t.Errorf("Login OAuth with selenium script failed with error %v and stderr %s", err, errBuffer.String())
	}
}

func LoginOAuthSelenium(t* testing.T, url string) {
	// We could use the go selenium library
	// But it does not support the latest selenium v4 just yet
	RunCommand(t, "python3", "../selenium_eduvpn.py", url)
}

func StateCallback(t *testing.T, oldState string, newState string, data string) {
	if newState == "OAuthInitialized" {
		LoginOAuthSelenium(t, data)
	}
}

func TestServer(t *testing.T) {
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
