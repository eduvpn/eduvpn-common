# Discovery
## Summary
Name: `Get Disco Servers` and `Get Disco Organizations`

Arguments: None

Returns: `JSON string for servers/organizations` and `Error for servers/organizations`

Note: Depending on the wrapper they may be combined into one function, the return value of this function is then the following:
`organizations, error for organizations, servers, errors for servers`

Used to obtain the servers and organizations list from the discovery server.
## Detailed information
Discovery is the aspect of eduVPN that allows a client to gather all the servers and organizations it can connect to. For this a discovery server is used, which is registered as `https://disco.eduvpn.org` in the library. We refer to the [official eduVPN documentation](https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md) to learn more about the exact way that these organizations and servers are structured.

The JSON data that this returns must be used by the client to build an UI. It is common for clients that the discovery functions get called on startup of the client. Note that there can be an error in retrieving the newest version of the servers/organizations. However, this library's goal is to ensure that a version is always available. Thus, a local copy is distributed with this library in the future.

This library also internally looks at the version of the servers and organizations such that rollbacks attacks are prevented. The client does not have to do any additional checks for this.

The structure of the JSON data is the structure in the [official eduVPN documentation](https://github.com/eduvpn/documentation/blob/v3/SERVER_DISCOVERY.md) without the `v` (version) field. So, for example, the servers list has a possible JSON structure of this:

```json
[
    {
        "server_type": "institute_access",
        "base_url": "https://hku.eduvpn.nl/",
        "display_name": {
            "en-US": "Utrecht School of the Arts",
            "nl-NL": "Hogeschool voor de Kunsten Utrecht"
        },
        "keyword_list": "hku"
    },
    {
        "server_type": "secure_internet",
        "base_url": "https://eduvpn.rash.al/",
        "country_code": "AL",
        "support_contact": [
            "mailto:helpdesk@rash.al"
        ]
    }
]
```

And the organizations list has a possible JSON structure of the following:

```json
[
    {
      "display_name": {
        "nl": "SURFnet bv",
        "en": "SURFnet bv"
      },
      "org_id": "https://idp.surfnet.nl",
      "secure_internet_home": "https://nl.eduvpn.org/",
      "keyword_list": {
        "en": "SURFnet bv SURF konijn surf surfnet powered by",
        "nl": "SURFnet bv SURF konijn powered by"
      }
    }
]
```


