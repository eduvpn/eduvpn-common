package util

import (
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
)

func TestCalculateGateway(t *testing.T) {
	cases := []struct {
		in   string
		want string
		err  string
	}{
		// normal cases
		{
			in:   "10.10.10.5/24",
			want: "10.10.10.1",
			err:  "",
		},
		{
			in:   "10.10.10.130/25",
			want: "10.10.10.129",
			err:  "",
		},
		{
			in:   "fd42::5/112",
			want: "fd42::1",
			err:  "",
		},
		{
			in:   "5502:df9::/64",
			want: "5502:df9::1",
			err:  "",
		},
		// unrealistic scenario but we have to handle these!
		{
			in:   "5502:df9::0/128",
			want: "",
			err:  "IP network does not contain incremented IP: 5502:df9::1",
		},
		{
			in:   "5502:df9::ffff/128",
			want: "",
			err:  "IP network does not contain incremented IP: 5502:df9::1:0",
		},
		{
			in:   "10.0.0.0/32",
			want: "",
			err:  "IP network does not contain incremented IP: 10.0.0.1",
		},
		{
			in:   "10.0.0.255/32",
			want: "",
			err:  "IP network does not contain incremented IP: 10.0.1.0",
		},
		// parsing errors
		{
			in:   "10.0.0.1",
			want: "",
			err:  "An internal error occurred. The cause of the error is: invalid CIDR address: 10.0.0.1.",
		},
		{
			in:   "bla",
			want: "",
			err:  "An internal error occurred. The cause of the error is: invalid CIDR address: bla.",
		},
		{
			in:   "5502:df9::ffff",
			want: "",
			err:  "An internal error occurred. The cause of the error is: invalid CIDR address: 5502:df9::ffff.",
		},
	}

	for _, c := range cases {
		got, err := CalculateGateway(c.in)
		test.AssertError(t, err, c.err)
		if got != c.want {
			t.Fatalf("got: %v not equal to want: %v", got, c.want)
		}
	}
}
