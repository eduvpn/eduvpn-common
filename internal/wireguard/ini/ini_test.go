package ini

import (
	"reflect"
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
)

func TestShouldSkip(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{
			in:   "test",
			want: false,
		},
		{
			in:   "#test",
			want: true,
		},
		{
			in:   "",
			want: true,
		},
	}

	for _, c := range cases {
		g := shouldSkip(c.in)
		if g != c.want {
			t.Fatalf("got: %v, not equal to want: %v", g, c.want)
		}
	}
}

func TestIsSection(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{
			in:   "[test]",
			want: true,
		},
		{
			in:   "#test",
			want: false,
		},
		{
			in:   "key=val",
			want: false,
		},
		// sections with empty names will be ignored later
		{
			in:   "[]",
			want: true,
		},
	}

	for _, c := range cases {
		g := isSection(c.in)
		if g != c.want {
			t.Fatalf("got: %v, not equal to want: %v", g, c.want)
		}
	}
}

func TestSectionName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{
			in:   "[test]",
			want: "test",
		},
		{
			in:   "[            spaces ]",
			want: "spaces",
		},
		{
			in:   "[]",
			want: "",
		},
		{
			in:   "",
			want: "",
		},
	}

	for _, c := range cases {
		g := sectionName(c.in)
		if g != c.want {
			t.Fatalf("got: %v, not equal to want: %v", g, c.want)
		}
	}
}

func TestKeyValue(t *testing.T) {
	cases := []struct {
		in      string
		wantk   string
		wantv   string
		wanterr string
	}{
		{
			in:      "bla",
			wantk:   "",
			wantv:   "",
			wanterr: "no key/value found",
		},
		{
			in:      "foo=bar",
			wantk:   "foo",
			wantv:   "bar",
			wanterr: "",
		},
		{
			in:      " foo   = bar ",
			wantk:   "foo",
			wantv:   "bar",
			wanterr: "",
		},
		{
			in:      "foo   = bar",
			wantk:   "foo",
			wantv:   "bar",
			wanterr: "",
		},
		{
			in:      "",
			wantk:   "",
			wantv:   "",
			wanterr: "no key/value found",
		},
		{
			in:      "=",
			wantk:   "",
			wantv:   "",
			wanterr: "key cannot be empty",
		},
		{
			in:      "empty=",
			wantk:   "empty",
			wantv:   "",
			wanterr: "",
		},
	}

	for _, c := range cases {
		gk, gv, gerr := keyValue(c.in)
		test.AssertError(t, gerr, c.wanterr)
		if gk != c.wantk {
			t.Fatalf("key, got: %v, not equal to want: %v", gk, c.wantk)
		}
		if gv != c.wantv {
			t.Fatalf("value, got: %v, not equal to want: %v", gv, c.wantv)
		}
	}
}

func TestOrderedKeysFind(t *testing.T) {
	cases := []struct {
		v  OrderedKeys
		in string
		w  int
	}{
		{
			v:  []string{""},
			in: "test",
			w:  -1,
		},
		{
			v:  []string{"bla"},
			in: "bla",
			w:  0,
		},
		{
			v:  []string{"ha"},
			in: "bla",
			w:  -1,
		},
		{
			v:  []string{"ha", "ga"},
			in: "ga",
			w:  1,
		},
	}

	for _, c := range cases {
		g := c.v.find(c.in)
		if g != c.w {
			t.Fatalf("got: %v, want: %v", g, c.w)
		}
	}
}

func TestOrderedKeysRemove(t *testing.T) {
	cases := []struct {
		v   OrderedKeys
		rem string
		out OrderedKeys
	}{
		{
			v:   []string{"bla"},
			rem: "test",
			out: []string{"bla"},
		},
		{
			v:   []string{"bla"},
			rem: "bla",
			out: []string{},
		},
		{
			v:   []string{"ha", "ga"},
			rem: "ga",
			out: []string{"ha"},
		},
	}

	for _, c := range cases {
		c.v.Remove(c.rem)
		if !reflect.DeepEqual(c.v, c.out) {
			t.Fatalf("got: %v, want: %v", c.v, c.out)
		}
	}
}

func TestParse(t *testing.T) {
	// parse correct file

	cases := []struct {
		in   string
		want INI
	}{
		{
			in:   ``,
			want: INI{},
		},
		{
			in: `
[section1]
bla=val
`,
			want: INI{
				sections: map[string]*Section{
					"section1": {
						keyValues: map[string]string{
							"bla": "val",
						},
						keys: []string{"bla"},
					},
				},
				keys: []string{"section1"},
			},
		},

		{
			in: `
# Portal: https://vpn.tuxed.net/vpn-user-portal/
# Profile: Default (default)
# Expires= 2025-01-23T15:56:58+00:00

[Interface]
MTU = 1392
PrivateKey = wowsoprivate=
Address = 10.142.221.3/24,fdb6:645e:c74e:a648::3/64
DNS = 9.9.9.9,2620:fe::fe

[Peer]
PublicKey = whydidimockthisitspublic=
AllowedIPs = 0.0.0.0/0,::/0
Endpoint = vpn.example.org:443
`,
			want: INI{
				sections: map[string]*Section{
					"Interface": {
						keyValues: map[string]string{
							"MTU":        "1392",
							"PrivateKey": "wowsoprivate=",
							"Address":    "10.142.221.3/24,fdb6:645e:c74e:a648::3/64",
							"DNS":        "9.9.9.9,2620:fe::fe",
						},
						keys: []string{"MTU", "PrivateKey", "Address", "DNS"},
					},
					"Peer": {
						keyValues: map[string]string{
							"PublicKey":  "whydidimockthisitspublic=",
							"AllowedIPs": "0.0.0.0/0,::/0",
							"Endpoint":   "vpn.example.org:443",
						},
						keys: []string{"PublicKey", "AllowedIPs", "Endpoint"},
					},
				},
				keys: []string{"Interface", "Peer"},
			},
		},
		{
			in: `
# Portal: https://vpn.tuxed.net/vpn-user-portal/
# Profile: Default (default)
# Expires= 2025-01-23T15:56:58+00:00

MTU = 1392
PrivateKey = wowsoprivate=
Address = 10.142.221.3/24,fdb6:645e:c74e:a648::3/64
DNS = 9.9.9.9,2620:fe::fe

PublicKey = whydidimockthisitspublic=
AllowedIPs = 0.0.0.0/0,::/0
Endpoint = vpn.example.org:443
`,
			want: INI{},
		},
	}

	for i, v := range cases {
		g := Parse(v.in)

		if !reflect.DeepEqual(g, v.want) {
			t.Fatalf("failed deep equal case %d, got: %#v, want: %#v", i, g, v.want)
		}
	}
}
