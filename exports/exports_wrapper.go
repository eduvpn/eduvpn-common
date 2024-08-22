package main

/*
#include "exports.h"

extern int test_state_callback(int old, int new, char* data);
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
	"github.com/eduvpn/eduvpn-common/types/error"
	"github.com/eduvpn/eduvpn-common/util"
)

func getString(in *C.char) string {
	if in == nil {
		return ""
	}
	defer FreeString(in)
	return C.GoString(in)
}

func getError(t *testing.T, gerr *C.char) string {
	jsonErr := getString(gerr)
	var transl err.Error

	if jsonErr == "" {
		return ""
	}

	jerr := json.Unmarshal([]byte(jsonErr), &transl)
	if jerr != nil {
		t.Fatalf("failed getting error JSON, val: %v, err: %v", jsonErr, jerr)
	}

	return util.GetLanguageMatched(transl.Message, "en")
}

//export test_state_callback
func test_state_callback(_ C.int, new C.int, data *C.char) int32 {
	if int(new) == 3 {
		fakeBrowserAuth(C.GoString(data)) //nolint:errcheck
		return 1
	}
	return 0
}

func testDoRegister(t *testing.T) string {
	nameS := C.CString("org.letsconnect-vpn.app.linux")
	defer FreeString(nameS)
	versionS := C.CString("0.0.1")
	defer FreeString(versionS)
	dir, err := os.MkdirTemp(os.TempDir(), "eduvpn-common-test-cgo")
	if err != nil {
		t.Fatalf("failed creating temp dir for state file: %v", err)
	}
	defer os.RemoveAll(dir)

	dirS := C.CString(dir)
	defer FreeString(dirS)

	return getError(t, Register(nameS, versionS, dirS, C.StateCB(C.test_state_callback), 0))
}

func mustRegister(t *testing.T) {
	err := testDoRegister(t)
	if err != "" {
		t.Fatalf("got register error: %v", err)
	}
}

func testRegister(t *testing.T) {
	mustRegister(t)
	defer Deregister()
	err := testDoRegister(t)
	if err == "" {
		t.Fatalf("got no register error after double registering: %v", err)
	}
}

func fakeBrowserAuth(str string) (string, error) {
	go func() {
		u, err := url.Parse(str)
		if err != nil {
			panic(err)
		}
		ru, err := url.Parse(u.Query().Get("redirect_uri"))
		if err != nil {
			panic(err)
		}
		oq := u.Query()
		q := ru.Query()
		q.Set("state", oq.Get("state"))
		q.Set("code", "fakeauthcode")
		ru.RawQuery = q.Encode()

		c := http.Client{}
		req, err := http.NewRequest("GET", ru.String(), nil)
		if err != nil {
			panic(err)
		}
		_, err = c.Do(req)
		if err != nil {
			panic(err)
		}
	}()
	return "", nil
}

func testServer(t *testing.T) *test.Server {
	// TODO: duplicate code between this and internal/api/api_test.go
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to setup listener for test server: %v", err)
	}

	hps := []test.HandlerPath{
		{
			Method: http.MethodGet,
			Path:   "/.well-known/vpn-user-portal",
			Response: fmt.Sprintf(`
{
  "api": {
    "http://eduvpn.org/api#3": {
      "api_endpoint": "https://%[1]s/test-api-endpoint",
      "authorization_endpoint": "https://%[1]s/test-authorization-endpoint",
      "token_endpoint": "https://%[1]s/test-token-endpoint"
    }
  },
  "v": "0.0.1"
}`, listen.Addr().String()),
		},
		{
			Method: http.MethodPost,
			Path:   "/test-token-endpoint",
			Response: `
{
	"access_token": "validaccess",
	"refresh_token": "validrefresh",
	"expires_in": 3600
}`,
		},
	}
	return test.NewServerWithHandles(hps, listen)
}

func testServerList(t *testing.T) {
	mustRegister(t)
	defer Deregister()
	serv := testServer(t)
	defer serv.Close()

	ck := CookieNew()
	defer CookieDelete(ck)

	list := fmt.Sprintf("https://%s", serv.Listener.Addr().String())
	listS := C.CString(list)
	defer FreeString(listS)

	sclient, err := serv.Client()
	if err != nil {
		t.Fatalf("failed to obtain server client: %v", err)
	}

	// TODO: can we do this better
	http.DefaultTransport = sclient.Client.Transport

	gerr := getError(t, AddServer(ck, 3, listS, nil))
	if gerr != "" {
		t.Fatalf("error adding server: %v", gerr)
	}

	glist, glistErr := ServerList()
	glistErrS := getError(t, glistErr)
	if glistErrS != "" {
		t.Fatalf("error getting server list: %v", glistErrS)
	}

	srvlistS := getString(glist)
	want := fmt.Sprintf(`{"custom_servers":[{"display_name":{"en":"127.0.0.1"},"identifier":"%s/","profiles":{"current":""}}]}`, list)
	if srvlistS != want {
		t.Fatalf("server list not equal, want: %v, got: %v", want, srvlistS)
	}

	remErr := getError(t, RemoveServer(3, listS))
	if remErr != "" {
		t.Fatalf("got error removing server: %v", remErr)
	}
	remErr = getError(t, RemoveServer(3, listS))
	if remErr == "" {
		t.Fatalf("got no error removing server again")
	}

	glist, glistErr = ServerList()
	glistErrS = getError(t, glistErr)
	if glistErrS != "" {
		t.Fatalf("error getting server list: %v", glistErrS)
	}

	srvlistS = getString(glist)
	want = "{}"
	if srvlistS != want {
		t.Fatalf("server list not equal, want: %v, got: %v", want, srvlistS)
	}
}
