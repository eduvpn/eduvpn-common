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
def register(self, debug: bool = False) -> None
```
- `debug`: Whether or not we want to enable debugging, default: `False`

Returns nothing. Raises an exception in case of an error.

## Discovery
See [Overview](../overview/discovery.html)
```python
def get_disco_servers(self) -> str
```
```python
def get_disco_organizations(self) -> str
```

Returns a `string` of JSON data with the servers/organizations. Raises an exception in case of an error.

## OpenVPN/Wireguard config
See [Overview](../overview/getconfig.html)
```python
def get_config_institute_access(self, url: str, forceTCP: bool = False) -> Tuple[str, str]
```
```python
def get_config_secure_internet(self, url: str, forceTCP: bool = False) -> Tuple[str, str]
```
- `url`: The url of the Secure Internet or Institute Access server to get a connect config for
- `forceTCP`: Whether or not we want to force enable TCP, default: `False`

Returns:
- A `string` of the OpenVPN/Wireguard config
- An `string`, `openvpn` or `wireguard` indicating if it is an OpenVPN or Wireguard config

Raises an exception in case of an error.

### Cancelling OAuth
```python
def cancel_oauth(self) -> None
```

Returns nothing. Raises an exception in case of an error.

### Setting a profile ID
```python
def set_profile(self, profile_id: str) -> None
```
- `profile_id`: The profile ID to connect to

Returns nothing. Raises an exception in case of an errorr.

## Connecting/Disconnecting
See [Overview](../overview/connecting.html)
```python
def set_connected(self) -> None
```
```python
def set_disconnected(self) -> None
```

Returns an nothing. Raises an exception in case of an error.

## Deregister
See [Overview](../overview/deregistering.html)
```python
def deregister() -> None
```

Returns nothing. Raises an exception in case of an error.

# Note on Callbacks
Some functions (e.g. [the API for getting an OpenVPN/Wireguard config](http://localhost:3000/api/overview/getconfig.html)) need a (or multiple) callbacks set. In Python, the callback function is set using decorators.
For this, the `eduvpn.EduVPN` class has the following syntax:

```python
# Where _eduvpn is the eduvpn.EduVPN class instance
# This gets called when the New_State_Example state is entered
# old_state is then the old state
@_eduvpn.event.on("New_State_Example", eduvpn.StateType.Enter)
def example_enter(old_state: str, data: str)
```
```python
# Where _eduvpn is the eduvpn.EduVPN class instance
# This gets called when the New_State_Example state is left
# new_state is then the new state
@_eduvpn.event.on("New_State_Example", eduvpn.StateType.Leave)
def example_leave(new_state: str, data: str)
```
To show how this can be done in practice, we will give an example in the next section.
