package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/eduvpn/eduvpn-common/internal/api/profiles"
	httpw "github.com/eduvpn/eduvpn-common/internal/http"
	"github.com/eduvpn/eduvpn-common/internal/test"
	"github.com/eduvpn/eduvpn-common/internal/wireguard"
	"github.com/eduvpn/eduvpn-common/types/protocol"
	"github.com/eduvpn/eduvpn-common/types/server"
	"github.com/jwijenbergh/eduoauth-go"
)

func tokenHandler(t *testing.T, gt []string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid HTTP method for token handler: %v", r.Method)
		}
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading token endpoint body: %v", err)
		}
		parsed, err := url.ParseQuery(string(b))
		if err != nil {
			t.Fatalf("failed parsing query body: %v", err)
		}
		grant := parsed.Get("grant_type")

		for _, v := range gt {
			if v == grant {
				_, err = w.Write([]byte(`
{
	"access_token": "validaccess",
	"refresh_token": "validrefresh",
	"expires_in": 3600
}
		`))
				if err != nil {
					t.Fatalf("failed writing in token handler: %v", err)
				}
				return
			}
		}
		t.Fatalf("grant type: %v, not allowed", grant)
	}
}

func checkAuthBearer(t *testing.T, r *http.Request) {
	authh := r.Header.Get("Authorization")
	if !strings.HasPrefix(authh, "Bearer ") {
		t.Fatalf("API call is not given with an authorization Bearer header, got: %v", authh)
	}
}

func connectHandler(t *testing.T, proto string, exp time.Time) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid HTTP method for connect handler: %v", r.Method)
		}
		checkAuthBearer(t, r)
		w.Header().Set("expires", exp.Format(http.TimeFormat))
		w.Header().Set("content-type", fmt.Sprintf("application/x-%s-profile", proto))
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed reading token endpoint body: %v", err)
		}
		parsed, err := url.ParseQuery(string(b))
		if err != nil {
			t.Fatalf("failed parsing query body: %v", err)
		}
		// the wireguard config we parse
		var cfg string
		if proto == "openvpn" {
			cfg = "openvpnconfig"
		} else {
			if parsed.Get("public_key") == "" {
				t.Fatalf("no public_key given")
			}
			if proto == "wireguard+tcp" {
				ptcp := parsed.Get("prefer_tcp")
				if ptcp != "yes" {
					t.Fatalf("prefer TCP is not yes: %s", ptcp)
				}
				cfg = `
[Interface]
[Peer]
ProxyEndpoint = https://proxyendpoint
`
			} else {
				cfg = "[Interface]"
			}
		}
		_, err = w.Write([]byte(cfg))
		if err != nil {
			t.Fatalf("failed writing /connect response: %v", err)
		}
	}
}

func disconnectHandler(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(_ http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("invalid HTTP method for disconnect handler: %v", r.Method)
		}
		checkAuthBearer(t, r)
	}
}

type TestCallback struct {
	t *testing.T
}

func (tc *TestCallback) TriggerAuth(_ context.Context, str string, _ bool) (string, error) {
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
func (tc *TestCallback) AuthDone(string, server.Type)                      {}
func (tc *TestCallback) TokensUpdated(string, server.Type, eduoauth.Token) {}

// create a API struct with allowed grant types
func createTestAPI(t *testing.T, tok *eduoauth.Token, gt []string, hps []test.HandlerPath) (*API, *test.Server) {
	// Create a simple API client and check if the fields are created correctly
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to setup listener for test server: %v", err)
	}

	hps = append(hps, []test.HandlerPath{
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
  "v": "0.0.0"
}					
`, listen.Addr().String()),
			ResponseCode: 200,
		},
		{
			Path:            "/test-token-endpoint",
			ResponseHandler: tokenHandler(t, gt),
		},
	}...)
	// start server
	serv := test.NewServerWithHandles(hps, listen)
	servc, err := serv.Client()
	if err != nil {
		t.Fatalf("failed to setup HTTP test server client: %v", servc)
	}

	sd := ServerData{
		ID:         "randomidentifier",
		Type:       server.TypeCustom,
		BaseWK:     serv.URL,
		BaseAuthWK: serv.URL,
		ProcessAuth: func(ctx context.Context, in string) (string, error) {
			return in, nil
		},
		DisableAuthorize: false,
		Transport:        servc.Client.Transport,
	}

	tc := &TestCallback{t: t}

	a, err := NewAPI(context.Background(), "testclient", sd, tc, tok)
	if err != nil {
		t.Fatalf("failed creating API: %v", err)
	}
	return a, serv
}

func TestNewAPI(t *testing.T) {
	gts := []string{"refresh_token"}
	tok := &eduoauth.Token{
		Access:  "expiredaccess",
		Refresh: "expiredrefresh",
		// tokens are expired, let's try authorizing
		ExpiredTimestamp: time.Now(),
	}
	a, srv := createTestAPI(t, tok, gts, nil)
	srv.Close()

	// now the tokens should be the new access tokens
	if a.oauth.Token().Access != "validaccess" {
		t.Fatalf("access token is not valid access")
	}
	if a.oauth.Token().Refresh != "validrefresh" {
		t.Fatalf("refresh token is not valid refresh")
	}

	gts = []string{"authorization_code"}
	tok = &eduoauth.Token{
		Access:           "expiredaccess",
		Refresh:          "",
		ExpiredTimestamp: time.Now(),
	}
	a, srv = createTestAPI(t, tok, gts, nil)
	srv.Close()

	// now the tokens should be the new access tokens
	if a.oauth.Token().Access != "validaccess" {
		t.Fatalf("access token is not valid access")
	}
	if a.oauth.Token().Refresh != "validrefresh" {
		t.Fatalf("refresh token is not valid refresh")
	}
}

func TestAPIInfo(t *testing.T) {
	// auth should not be triggered
	var gts []string
	tok := &eduoauth.Token{
		Access:           "validaccess",
		Refresh:          "validrefresh",
		ExpiredTimestamp: time.Now().Add(1 * time.Hour),
	}
	statErr := &httpw.StatusError{}
	cases := []struct {
		hp   test.HandlerPath
		info *profiles.Info
		err  interface{}
	}{
		{
			hp: test.HandlerPath{
				Method: http.MethodGet,
				Path:   "/test-api-endpoint/info",
				Response: `
{
    "info": {
        "profile_list": [
            {
                "default_gateway": false,
                "display_name": "test profile 1",
                "profile_id": "test1",
                "vpn_proto_list": [
                    "openvpn",
                    "wireguard"
                ]
            }
        ]
    }
}
`,
				ResponseCode: 200,
			},
			info: &profiles.Info{
				Info: profiles.ListInfo{
					ProfileList: []profiles.Profile{
						{
							ID:             "test1",
							DisplayName:    "test profile 1",
							VPNProtoList:   []string{"openvpn", "wireguard"},
							DefaultGateway: false,
						},
					},
				},
			},
		},
		{
			hp: test.HandlerPath{
				Method: http.MethodGet,
				Path:   "/test-api-endpoint/info",
				Response: `
{
    "info": {
        "profile_list": [
            {
                "display_name": "test profile 2",
                "profile_id": "test2",
                "vpn_proto_list": [
                    "wireguard"
                ]
            }
        ]
    }
}
`,
				ResponseCode: 200,
			},
			info: &profiles.Info{
				Info: profiles.ListInfo{
					ProfileList: []profiles.Profile{
						{
							ID:             "test2",
							DisplayName:    "test profile 2",
							VPNProtoList:   []string{"wireguard"},
							DefaultGateway: false,
						},
					},
				},
			},
		},
		{
			hp: test.HandlerPath{
				Method:       http.MethodGet,
				Path:         "/test-api-endpoint/info",
				Response:     "",
				ResponseCode: 404,
			},
			info: nil,
			err:  &statErr,
		},
	}

	for _, c := range cases {
		a, srv := createTestAPI(t, tok, gts, []test.HandlerPath{c.hp})
		defer srv.Close()
		gprfs, err := a.Info(context.Background())
		// got error but the want error is nil
		if err != nil {
			if c.err == nil {
				t.Fatalf("failed profiles info: %v but want no error", err)
			}

			if !errors.As(err, c.err) {
				t.Fatalf("error type not equal: %T, want: %T, error string: %s", err, c.err, err.Error())
			}
		} else if c.err != nil {
			t.Fatalf("got no error but want error: %T", c.err)
		}

		if !reflect.DeepEqual(gprfs, c.info) {
			t.Fatalf("got info: %v, not equal to want: %v", gprfs, c.info)
		}
	}
}

func TestAPIConnect(t *testing.T) {
	// auth should not be triggered
	var gts []string
	tok := &eduoauth.Token{
		Access:           "validaccess",
		Refresh:          "validrefresh",
		ExpiredTimestamp: time.Now().Add(1 * time.Hour),
	}
	cases := []struct {
		hp     test.HandlerPath
		cd     *ConnectData
		prof   profiles.Profile
		protos []protocol.Protocol
		ptcp   bool
		err    error
	}{
		{
			hp: test.HandlerPath{
				Method:       http.MethodPost,
				Path:         "/test-api-endpoint/connect",
				Response:     ``,
				ResponseCode: 200,
			},
			cd:  nil,
			err: ErrNoProtocols,
		},
		{
			hp: test.HandlerPath{
				Method:       http.MethodPost,
				Path:         "/test-api-endpoint/connect",
				Response:     ``,
				ResponseCode: 200,
			},
			cd:     nil,
			protos: []protocol.Protocol{protocol.Unknown},
			err:    ErrUnknownProtocol,
		},
		{
			hp: test.HandlerPath{
				Method:       http.MethodPost,
				Path:         "/test-api-endpoint/connect",
				Response:     ``,
				ResponseCode: 200,
			},
			cd:     nil,
			protos: []protocol.Protocol{protocol.OpenVPN, protocol.WireGuard, protocol.Unknown},
			err:    ErrUnknownProtocol,
		},
		{
			hp: test.HandlerPath{
				Method:          http.MethodPost,
				Path:            "/test-api-endpoint/connect",
				ResponseHandler: connectHandler(t, "openvpn", time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC)),
			},
			cd: &ConnectData{
				Configuration: "openvpnconfig\nscript-security 0",
				Protocol:      protocol.OpenVPN,
				Expires:       time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC),
				Proxy:         nil,
			},
			protos: []protocol.Protocol{protocol.OpenVPN, protocol.WireGuard},
			err:    nil,
		},
		{
			hp: test.HandlerPath{
				Method:          http.MethodPost,
				Path:            "/test-api-endpoint/connect",
				ResponseHandler: connectHandler(t, "wireguard", time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC)),
			},
			cd: &ConnectData{
				Configuration: `\[Interface\]
PrivateKey = .*`,
				Protocol: protocol.WireGuard,
				Expires:  time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC),
				Proxy:    nil,
			},
			protos: []protocol.Protocol{protocol.OpenVPN, protocol.WireGuard},
			err:    nil,
		},
		{
			hp: test.HandlerPath{
				Method:          http.MethodPost,
				Path:            "/test-api-endpoint/connect",
				ResponseHandler: connectHandler(t, "wireguard+tcp", time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC)),
			},
			cd: &ConnectData{
				Configuration: `\[Interface\]
PrivateKey = .*`,
				Protocol: protocol.WireGuardProxy,
				Expires:  time.Date(2000, time.January, 0, 0, 0, 0, 0, time.UTC),
				// proxy will be manually checked
				Proxy: &wireguard.Proxy{},
			},
			ptcp:   true,
			protos: []protocol.Protocol{protocol.OpenVPN, protocol.WireGuard},
			err:    nil,
		},
	}

	for _, c := range cases {
		a, srv := createTestAPI(t, tok, gts, []test.HandlerPath{c.hp})
		defer srv.Close()
		gcd, err := a.Connect(context.Background(), c.prof, c.protos, c.ptcp)
		// got error but the want error is nil
		if err != nil {
			if c.err == nil {
				t.Fatalf("failed connect: %v but want no error", err)
			}

			if !errors.Is(err, c.err) {
				t.Fatalf("error type not equal: %T, want: %T, error string: %s", err, c.err, err)
			}
		} else if c.err != nil {
			t.Fatalf("got no error but want error: %T", c.err)
		}

		if gcd != nil && c.cd != nil {
			m, err := regexp.MatchString(c.cd.Configuration, gcd.Configuration)
			if err != nil {
				t.Fatalf("failed matching regexp: %v", err)
			}
			if !m {
				t.Fatalf("regex:\n%s\ndoes not match config:\n%s", c.cd.Configuration, gcd.Configuration)
			}
			// we have already checked the config using a regex
			c.cd.Configuration = gcd.Configuration

			// check proxy manually
			if c.cd.Proxy != nil && gcd.Proxy != nil {
				if gcd.Proxy.Peer != "https://proxyendpoint" {
					t.Fatalf("config data proxy peer is no proxyendpoint with HTTPS scheme: %s", gcd.Proxy.Peer)
				}
				if gcd.Proxy.SourcePort <= 0 {
					t.Fatalf("got proxy source port is smaller or equal to 0: %v", gcd.Proxy.SourcePort)
				}
				if !strings.Contains(gcd.Proxy.Listen, "127.0.0.1") {
					t.Fatalf("proxy listen does not contain 127.0.0.1: %s", gcd.Proxy.Listen)
				}
				c.cd.Proxy = gcd.Proxy
			}
		}
		if !reflect.DeepEqual(gcd, c.cd) {
			t.Fatalf("got connect data: %v, not equal to want: %v", gcd, c.cd)
		}
	}
}

func TestDisconnect(t *testing.T) {
	var gts []string
	tok := &eduoauth.Token{
		Access:           "validaccess",
		Refresh:          "validrefresh",
		ExpiredTimestamp: time.Now().Add(1 * time.Hour),
	}
	a, srv := createTestAPI(t, tok, gts, []test.HandlerPath{
		{
			Path:            "/test-api-endpoint/disconnect",
			ResponseHandler: disconnectHandler(t),
		},
	})
	defer srv.Close()
	err := a.Disconnect(context.Background())
	if err != nil {
		t.Fatalf("failed /disconnect: %v", err)
	}
}
