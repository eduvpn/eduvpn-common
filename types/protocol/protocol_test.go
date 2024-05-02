package protocol

import "testing"

func TestNew(t *testing.T) {
	cases := []struct {
		in   string
		want Protocol
	}{
		{
			in:   "openvpn",
			want: OpenVPN,
		},
		{
			in:   "wireguard",
			want: WireGuard,
		},
		{
			in:   "wrong",
			want: Unknown,
		},
		{
			in:   "",
			want: Unknown,
		},
	}

	for _, c := range cases {
		got := New(c.in)
		if got != c.want {
			t.Fatalf("got: %v, not equal to want: %v", got, c.want)
		}
	}
}
