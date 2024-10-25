//go:build cgotesting

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
	"regexp"
	"strings"
	"testing"
	"time"

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

// ClonedAskTransition is a clone of the struct types/server.go RequiredAskTransition
// It is cloned here to ensure that when the types API changes, the tests have to be changed as well
type ClonedAskTransition struct {
	Cookie int         `json:"cookie"`
	Data   interface{} `json:"data"`
}

//export test_state_callback
func test_state_callback(_ C.int, new C.int, data *C.char) int32 {
	// OAUTH_STARTED
	// We use hardcoded values here instead of constants
	// to ensure that a change in the API needs to be changed here too
	if int(new) == 3 {
		fakeBrowserAuth(C.GoString(data)) //nolint:errcheck
		return 1
	}
	// ASK_PROFILE
	if int(new) == 6 {
		dataS := C.GoString(data)
		var tr ClonedAskTransition
		jsonErr := json.Unmarshal([]byte(dataS), &tr)
		if jsonErr != nil {
			panic(jsonErr)
		}
		prS := C.CString("employees")
		defer FreeString(prS)
		CookieReply(C.uint64_t(tr.Cookie), prS)
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
		{
			Method: http.MethodGet,
			Path:   "/test-api-endpoint/info",
			Response: `

{
    "info": {
        "profile_list": [
            {
                "default_gateway": true,
                "display_name": "Employees",
                "profile_id": "employees",
                "vpn_proto_list": [
                    "openvpn",
                    "wireguard"
                ]
            },
            {
                "default_gateway": true,
                "display_name": "Other",
                "profile_id": "other",
                "vpn_proto_list": [
                    "openvpn",
                    "wireguard"
                ]
            }
        ]
    }
}`,
		},
		{
			Method: http.MethodPost,
			Path:   "/test-api-endpoint/disconnect",
		},
		{
			Path: "/test-api-endpoint/connect",
			ResponseHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}
				w.Header().Add("expires", time.Now().Add(4*time.Hour).Format(http.TimeFormat))
				w.Header().Add("Content-Type", "application/x-wireguard-profile")
				w.WriteHeader(200)
				// example from https://docs.eduvpn.org/server/v3/api.html#response_1
				resp := `
Expires: Fri, 06 Aug 2021 03:59:59 GMT
Content-Type: application/x-wireguard-profile

[Interface]
Address = 10.43.43.2/24, fd43::2/64
DNS = 9.9.9.9, 2620:fe::fe

[Peer]
PublicKey = iWAHXts9w9fQVEbA5pVriPlAYMwwEPD5XcVCZDZn1AE=
AllowedIPs = 0.0.0.0/0, ::/0
Endpoint = vpn.example:51820`
				_, err := w.Write([]byte(resp))
				if err != nil {
					panic(err)
				}
			},
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
	defer CookieCancel(ck)

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

// ClonedExpiryTimes is a copy of types/server Expiry
// to ensure that when the public API is changed, this should be changed too
type ClonedExpiryTimes struct {
	// StartTime is the start time of the VPN in Unix
	StartTime int64 `json:"start_time"`
	// EndTime is the end time of the VPN in Unix.
	EndTime int64 `json:"end_time"`
	// ButtonTime is the Unix time at which to start showing the renew button in the UI
	ButtonTime int64 `json:"button_time"`
	// CountdownTime is the Unix time at which to start showing more detailed countdown timer.
	// E.g. first start with days (7 days left), and when the current time is after this time, show e.g. 9 minutes and 59 seconds
	CountdownTime int64 `json:"countdown_time"`
	// NotificationTimes is the slice/list of times at which to show a notification that the VPN is about to expire
	NotificationTimes []int64 `json:"notification_times"`
}

func testExpiryTimes(t *testing.T) {
	exp, expErr := ExpiryTimes()
	expErrS := getError(t, expErr)
	if expErrS != "" {
		t.Fatalf("expiry times error is not empty: %v", expErrS)
	}

	expS := getString(exp)

	var et ClonedExpiryTimes

	jErr := json.Unmarshal([]byte(expS), &et)
	if jErr != nil {
		t.Fatalf("failed parsing expiry times as JSON: %v", jErr)
	}
	etu := time.Unix(et.EndTime, 0)
	stu := time.Unix(et.StartTime, 0)

	between := func(label string, cand time.Time, equalS bool, equalE bool) {
		if !cand.After(stu) && (!equalS || !cand.Equal(stu)) {
			t.Fatalf("%s: %v, is not after start time: %v", label, cand, stu)
		}
		if !cand.Before(etu) && (!equalE || !cand.Equal(etu)) {
			t.Fatalf("%s: %v, is after end time: %v", label, cand, etu)
		}
	}

	now := time.Now()
	between("now", now, false, false)
	btu := time.Unix(et.ButtonTime, 0)
	between("button time", btu, false, false)
	ctu := time.Unix(et.CountdownTime, 0)
	between("countdown time", ctu, true, false)

	first := true
	for _, v := range et.NotificationTimes {
		curr := time.Unix(v, 0)
		between("notification time", curr, false, first)
		first = false
	}
}

func testSetProfileID(t *testing.T) {
	prfS := C.CString("idontexist")
	defer FreeString(prfS)
	pErr := getError(t, SetProfileID(prfS))
	// this might sound wrong but it's correct!
	// setting an incorrect profile should still work
	// as it can be used for migrating old profiles to the new common codebase
	if pErr != "" {
		t.Fatal("got empty error setting non-existent profile")
	}
	prfS2 := C.CString("employees")
	defer FreeString(prfS2)
	pErr = getError(t, SetProfileID(prfS2))
	if pErr != "" {
		t.Fatal("got error setting existent profile")
	}
}

func testRenewSession(t *testing.T) {
	ck := CookieNew()
	rErr := getError(t, RenewSession(ck))
	if rErr != "" {
		t.Fatalf("failed renewing session: %v", rErr)
	}
}

func testCleanup(t *testing.T) {
	ck := CookieNew()
	defer CookieDelete(ck)
	cErr := getError(t, Cleanup(ck))
	if cErr != "" {
		t.Fatalf("failed cleaning up connection: %v", cErr)
	}
}

func testGetConfig(t *testing.T) {
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

	_, cfgErr := GetConfig(ck, 3, listS, 0, 0)
	cfgErrS := getError(t, cfgErr)
	if !strings.HasSuffix(cfgErrS, "server does not exist.") {
		t.Fatalf("error does not end with 'server does not exist.': %v", cfgErrS)
	}

	// add the server
	addErr := getError(t, AddServer(ck, 3, listS, nil))
	if addErr != "" {
		t.Fatalf("failed to add server: %v", addErr)
	}

	cfg, cfgErr := GetConfig(ck, 3, listS, 0, 0)
	cfgErrS = getError(t, cfgErr)
	if cfgErrS != "" {
		t.Fatalf("failed to get config for server: %v", cfgErrS)
	}
	cfgS := getString(cfg)

	// match the config with the private key in the middle
	bRe := `{"config":"[Interface]\nAddress = 10.43.43.2/24, fd43::2/64\nDNS = 9.9.9.9, 2620:fe::fe\nPrivateKey = `
	aRe := `\n[Peer]\nPublicKey = iWAHXts9w9fQVEbA5pVriPlAYMwwEPD5XcVCZDZn1AE=\nAllowedIPs = 0.0.0.0/0, ::/0\nEndpoint = vpn.example:51820\n","protocol":2,"default_gateway":true,"should_failover":true}`

	// simple regex to match the key, see https://lists.zx2c4.com/pipermail/wireguard/2020-December/006222.html
	re := fmt.Sprintf("%s[A-Za-z0-9+/]{42}[AEIMQUYcgkosw480]=%s", regexp.QuoteMeta(bRe), regexp.QuoteMeta(aRe))
	ok, rErr := regexp.MatchString(re, cfgS)
	if rErr != nil {
		t.Fatalf("failed matching regexp: %v", rErr)
	}
	if !ok {
		t.Fatalf("VPN config does not match regex: %v", cfgS)
	}

	// 7 = GotConfig
	stateIn, statErr := InState(7)
	statErrS := getError(t, statErr)
	if statErrS != "" {
		t.Fatalf("got a state error when checking if client is in state: %v", statErr)
	}
	if stateIn == 0 {
		t.Fatal("client is not in State 7: GotConfig")
	}
	setState := func(in C.int) {
		// set state connecting
		statErr := getError(t, SetState(in))
		if statErr != "" {
			t.Fatalf("failed to set state: %v, err: %v", in, statErr)
		}
	}

	// set connecting -> connected -> disconnecting -> disconnected
	setState(8)
	setState(9)
	setState(10)
	setState(11)

	testExpiryTimes(t)
	testSetProfileID(t)
	testRenewSession(t)
	testCleanup(t)
}

func testLetsConnectDiscovery(t *testing.T) {
	// this registers a let's connect! client
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

	// try to add an institute access server
	exptErr := fmt.Sprintf("An internal error occurred. The cause of the error is: Adding a non-custom server when the client does not use discovery is not supported, identifier: %s, type: 1.", list)
	addErr := getError(t, AddServer(ck, 1, listS, nil))
	if addErr != exptErr {
		t.Fatalf("failed to add server got a different error: %v, want: %v", addErr, exptErr)
	}

	_, servErr := DiscoServers(ck, nil)
	servErrS := getError(t, servErr)
	exptErr = "An internal error occurred. The cause of the error is: Server discovery with this client ID is not supported."
	if servErrS != exptErr {
		t.Fatalf("discovery servers got a different error: %v, want: %v", servErrS, exptErr)
	}

	_, orgErr := DiscoOrganizations(ck, nil)
	orgErrS := getError(t, orgErr)
	exptErr = "An internal error occurred. The cause of the error is: Organization discovery with this client ID is not supported."
	if orgErrS != exptErr {
		t.Fatalf("discovery organizations got a different error: %v, want: %v", orgErrS, exptErr)
	}
}
