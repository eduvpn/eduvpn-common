# Functions
## Registering
See [Overview](../overview/registering.html)
```go
func Register(name string, directory string, stateCallback func, debug bool) error
```
- `name`: The name of the client
- `directory`: The directory where the configs and logging should be stored
- `stateCallback`: function with three arguments, full type:
  ```go
  func(oldState string, newState string, data string)
  ```
- `debug`: Whether or not we want to enable debugging

Returns an `error` type, nil if no error

## Discovery
See [Overview](../overview/discovery.html)
```go
func GetDiscoServers() (string, error)
func GetDiscoOrganizations() (string, error)
```

Returns a string of JSON data with the servers/organizations and an `error`, nil if no error

## OpenVPN/Wireguard config
See [Overview](../overview/getconfig.html)
```go
func GetConnectConfig(url string, forceTCP bool) (string, string, error)
```
- `url`: The url of the server to get a connect config for
- `forceTCP`: Whether or not we want to force enable TCP

Returns:
- A `string` of the OpenVPN/Wireguard config
- A `string`, `openvpn` or `wireguard` indicating if it is an OpenVPN or Wireguard config
- An `error` (can be nil)

### Setting a profile ID
```go
func SetProfileID(profileID string) error
```
- `profileID`: The profile ID to connect to

Returns an `error`, can be nil indicating no error

## Connecting/Disconnecting
See [Overview](../overview/connecting.html)
```go
func SetConnected() error
func SetDisconnected() error
```

Returns an `error`, can be nil indicating no error

## Deregister
See [Overview](../overview/deregistering.html)
```go
func Deregister() error
```

Returns an `error`, can be nil indicating no error
