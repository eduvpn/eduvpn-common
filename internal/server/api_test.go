package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
	"github.com/go-errors/errors"
)

func getErrorMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func compareEndpoints(ep1 Endpoints, ep2 Endpoints) bool {
	v3_1 := ep1.API.V3
	v3_2 := ep2.API.V3
	return v3_1.API == v3_2.API && v3_1.Authorization == v3_2.Authorization && v3_1.Token == v3_2.Token
}

func Test_APIGetEndpoints(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello!")
	})
	hs := &test.HandlerSet{}
	hs.SetHandler(handler)
	s := test.NewServer(hs)
	defer s.Close()

	c, err := s.Client()
	if err != nil {
		t.Fatalf("failed to get client for test server endpoints: %v", err)
	}

	testCases := []struct {
		epl EndpointList
		err error
	}{
		{
			epl: EndpointList{
				API:           "https://example.com/1",
				Authorization: "https://example.com/2",
				Token:         "https://example.com/3",
			},
			err: nil,
		},
		{
			epl: EndpointList{
				API:           "http://example.com/1",
				Authorization: "https://example.com/2",
				Token:         "https://example.com/3",
			},
			err: errors.New("API scheme: 'http', is not equal to 'https'"),
		},
		{
			epl: EndpointList{
				API:           "https://example.com/1",
				Authorization: "https://example.com/2",
				Token:         "ftp://example.com/3",
			},
			err: errors.New("API scheme: 'https', is not equal to token scheme: 'ftp'"),
		},
	}

	for _, tc := range testCases {
		ep := &Endpoints{
			API: EndpointsVersions{
				V3: tc.epl,
			},
		}
		// Update the handler
		hs.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			jsonStr, err := json.Marshal(ep)
			if err != nil {
				t.Fatalf("failed to marshal JSON for test case: %v, err: %v", tc, err)
			}

			fmt.Fprintln(w, string(jsonStr))
		}))
		gotEP, err := APIGetEndpoints(s.URL, c)
		if getErrorMsg(err) != getErrorMsg(tc.err) {
			t.Fatalf("Errors not equal, want err: %v, got: %v", tc.err, err)
		}
		// The error was not nil, continue because endpoints should not be compared
		if tc.err != nil {
			continue
		}
		if ep == nil {
			t.Fatalf("No test case endpoints")
		}
		if gotEP == nil {
			t.Fatalf("Got no endpoints for nil error")
		}
		// if no error then the endpoints should be equal
		if !compareEndpoints(*ep, *gotEP) {
			t.Fatalf("Endpoints are not equal, got: %v, want: %v", gotEP, ep)
		}
	}
}
