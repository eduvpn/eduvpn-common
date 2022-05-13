# Getting an OpenVPN/Wireguard config
## Summary
name: `Get Config Institute Access` and `Get Config Secure Internet`

| Arguments | Description                             | type     |
| --------- | --------------------------------------  | -------- |
| URL       | The url of the VPN server to connect to | string   |
| Force TCP | Whether or not to force the use of TCP  | string   |

Returns: `OpenVPN/Wireguard config (string)` `wireguard/openvpn type (string)`, `Error`

Used to obtain the OpenVPN/Wireguard config

## Detailed information

To get a configuration that is used to actually establish a tunnel with the VPN server, we have the Get Config function for Institute Access and Secure Internet (the exact name depends on the language you're using) function in the library. This function has two parameters *URL* and *Force TCP*.

*URL* is the base url of the server to connect to
e.g. `nl.eduvpn.org`. Use the correct function to indicate if it is an Institute Access server or a Secure Internet server. A user configured server is often an Institute Access server.

The *Force TCP* flag is a boolean that indicates whether or not we want to use TCP to connect over the VPN. This flag is useful if the user has enabled e.g. a setting that forces the use of TCP, which is only supported by OpenVPN. If the Force TCP flag is set to true but the server only supports Wireguard then an error is returned and the config will be empty.

This function takes care of OAuth which has certain callbacks with data. Additionally, there are also callbacks that need to be registered for selecting the right profile to connect to. These callbacks will be explained now.

The data that this function returns is the OpenVPN/Wireguard config as a string, the type of config (a string: "wireguard" or "openvpn") and an error if present.

### Callback: OAuth started

OAuth has an important callback which is used to obtain the authorization URL by the client. This client needs to open this authorization URL in a web browser such that the user can authenticate with the VPN portal and then authorize the client to obtain OpenVPN/Wireguard configs.

The callback for this is triggered whenever the OAuth Started state is triggered. The data which this callback has is the authorization url that needs to be opened in the web browser.

The format of the authorization URL is e.g. this:

`https://eduvpn.example.com/vpn-user-portal/oauth/authorize?client_id=org.eduvpn.app.linux&code_challenge=DsmGyWFBkvDXiIO33Fs40Z0fn4pxtzDCW2jKvAMptBg&code_challenge_method=S256&redirect_uri=http%3A%2F%2F127.0.0.1%3A8000%2Fcallback&response_type=code&scope=config&state=vha2Krx-HpOyvFkWsWYmey0jrHQ6bnb06PQ6zBXX_bg`

This callback can be cancelled by using a `Cancel OAuth` function.

### Callback: Selecting a profile

Another aspect that needs to be taken into account is the fact that there can be multiple profiles that a client can connect to. When the function gets called for obtaining an OpenVPN/Wireguard configuration, it asks the client which profile it wants to connect to using the callback that gets triggered on the Ask Profile state. The data is the list of profiles in JSON format, e.g.

```json
{
  "info": {
    "profile_list": [
      {
        "profile_id": "internet",
        "default_gateway": true,
        "display_name": "IPv4 (NAT) IPv6 (GUA) Access",
        "vpn_proto_list": [
          "openvpn"
        ]
      },
      {
        "profile_id": "adblock",
        "default_gateway": true,
        "display_name": "Malware/Tracking-Blocker IPv4 (NAT) IPv6 (GUA)",
        "vpn_proto_list": [
          "openvpn"
        ]
      },
      {
        "profile_id": "dnsonly",
        "default_gateway": false,
        "display_name": "DNS-Only & Malware/Tracking-Blocker (experimental)",
        "vpn_proto_list": [
          "openvpn"
        ]
      }
    ]
  }
}
```

For actually selecting the profile, there is a separate function which takes care of this. This function takes as only argument the profile ID as a string.
