# Functions
## Creating the class
See [Overview](../overview/registering.html)

This creates the class and basically forwards these arguments when `register` is called.
```python
def __init__(self, name: str, directory: str)
```
- `name`: The name of the client
- `directory`: The directory where the configs and logging should be stored

## Registering
See [Overview](../overview/registering.html)
```python
def register(self, debug=False: bool) -> Optional[str]
```
- `debug`: Whether or not we want to enable debugging

Returns an optional `string` for the error message

## Discovery
See [Overview](../overview/discovery.html)
```python
def get_disco_servers(self) -> (Optional[str], Optional[str])
```
```python
def get_disco_organizations(self) -> (Optional[str], Optional[str])
```

Returns an optional `string` of JSON data with the servers/organizations and an optional error message

## OpenVPN/Wireguard config
See [Overview](../overview/getconfig.html)
```python
def get_connect_config(self, url: str, forceTCP: bool) -> (Optional[str], Optional[str], Optional[str])
```
- `url`: The url of the server to get a connect config for
- `forceTCP`: Whether or not we want to force enable TCP

Returns:
- An optional `string` of the OpenVPN/Wireguard config
- An optional `string`, `openvpn` or `wireguard` indicating if it is an OpenVPN or Wireguard config
- An optional error message `string`

### Setting a profile ID
```python
def set_profile(self, profile_id: str) -> Optional[str]
```
- `profile_id`: The profile ID to connect to

Returns an optional `string`, which is the error message

## Connecting/Disconnecting
See [Overview](../overview/connecting.html)
```python
def set_connected(self) -> Optional[str]
```
```python
def set_disconnected(self) -> Optional[str]
```

Returns an optional `string`, which is the error message

## Deregister
See [Overview](../overview/deregistering.html)
```python
def deregister() -> Optional[str]
```

Returns an optional `string`, which is the error message
