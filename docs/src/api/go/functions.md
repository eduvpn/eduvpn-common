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
  func StateCallback(oldState string, newState string, data string)
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
func GetConfigInstituteAccess(url string, forceTCP bool) (string, string, error)
func GetConfigSecureInternet(url string, forceTCP bool) (string, string, error)
```
- `url`: The URL of the Institute Access or Secure Internet server to get a connect config for
- `forceTCP`: Whether or not we want to force enable TCP

Returns:
- A `string` of the OpenVPN/Wireguard config
- A `string`, `openvpn` or `wireguard` indicating if it is an OpenVPN or Wireguard config
- An `error` (can be nil)

### Cancelling OAuth
```go
func CancelOAuth() error
```
Returns an `error`, can be nil indicating no error

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

# Note on Callbacks
Some functions (e.g. [the API for getting an OpenVPN/Wireguard config](http://localhost:3000/api/overview/getconfig.html)) need a (or multiple) callback(s) set. In Go, the callback function is given in the [Register function](#registering). The signature of this function is the following:
```go
func StateCallback(oldState string, newState string, data string)
```
Because certain callbacks need to be set, you can simply compare against `oldState` and `newState`. To show how this can be done in practice, we will give an example in the next section.
