package types

// Shared server types

// Structs that define the json format for
// url: "https://disco.eduvpn.org/v2/organization_list.json"
type DiscoveryOrganizations struct {
	Version   uint64                  `json:"v"`
	List      []DiscoveryOrganization `json:"organization_list"`
	Timestamp int64                   `json:"-"`
	RawString string                  `json:"-"`
}

type DiscoveryOrganization struct {
	DisplayName struct {
		En string `json:"en"`
	} `json:"display_name"`
	OrgId              string `json:"org_id"`
	SecureInternetHome string `json:"secure_internet_home"`
	KeywordList        struct {
		En string `json:"en"`
	} `json:"keyword_list"`
}

// Structs that define the json format for
// url: "https://disco.eduvpn.org/v2/server_list.json"
type DiscoveryServers struct {
	Version   uint64            `json:"v"`
	List      []DiscoveryServer `json:"server_list"`
	Timestamp int64             `json:"-"`
	RawString string            `json:"-"`
}

type DiscoveryServer struct {
	AuthenticationURLTemplate string   `json:"authentication_url_template"`
	BaseURL                   string   `json:"base_url"`
	CountryCode               string   `json:"country_code"`
	PublicKeyList             []string `json:"public_key_list"`
	Type                      string   `json:"server_type"`
	SupportContact            []string `json:"support_contact"`
}
