package v2

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/eduvpn/eduvpn-common/internal/test"
	"github.com/eduvpn/eduvpn-common/types/server"
)

func TestLoad(t *testing.T) {
	cases := []struct {
		json    string
		want    *V2
		wantErr string
	}{
		// normal v2 config
		{
			json: `
{
    "server_list": {
	"1,a": {
	    "profiles": {
		"current": "a",
		"map": {
		    "a": {
			"display_name": {
			    "en": "a"
			}
		    }
		}
	    }
	}
    }
}
`,
			want: &V2{
				List: map[ServerKey]*Server{
					{ID: "a", T: server.TypeInstituteAccess}: {
						Profiles: server.Profiles{
							Map: map[string]server.Profile{
								"a": {DisplayName: map[string]string{"en": "a"}},
							},
							Current: "a",
						},
					},
				},
			},
			wantErr: "",
		},
		{
			json: `
{
    "server_list": {
	"a,1": {
	    "profiles": {
		"current": "a",
		"map": {
		    "a": {
			"display_name": {
			    "en": "a"
			}
		    }
		}
	    }
	}
    }
}
`,
			want:    nil,
			wantErr: "expected integer",
		},
		{
			json: `
{
    "server_list": {
	"1,a": {
	    "profiles": {
		"current": "a",
		"map": {
		    "a": {
			"display_name": {
			    "en": "a"
			}
		    }
		}
	    }
	},
	"2,a": {
	    "profiles": {
		"current": "a",
		"map": {
		    "a": {
			"display_name": {
			    "en": "a"
			}
		    }
		}
	    }
	}
    }
}
`,
			want: &V2{
				List: map[ServerKey]*Server{
					{ID: "a", T: server.TypeInstituteAccess}: {
						Profiles: server.Profiles{
							Map: map[string]server.Profile{
								"a": {DisplayName: map[string]string{"en": "a"}},
							},
							Current: "a",
						},
					},
					{ID: "a", T: server.TypeSecureInternet}: {
						Profiles: server.Profiles{
							Map: map[string]server.Profile{
								"a": {DisplayName: map[string]string{"en": "a"}},
							},
							Current: "a",
						},
					},
				},
			},
			wantErr: "",
		},
	}

	for _, v := range cases {
		var g *V2
		err := json.Unmarshal([]byte(v.json), &g)
		test.AssertError(t, err, v.wantErr)
		if err == nil {
			if !reflect.DeepEqual(g, v.want) {
				t.Fatalf("structs not equal")
			}
		}
	}
}
