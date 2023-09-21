This document was automatically generated from the exports/exports.go file

# Table of contents
- [About the API](#about-the-api)
- [Functions](#functions)
    * [AddServer](#addserver)
    * [Cleanup](#cleanup)
    * [CookieCancel](#cookiecancel)
    * [CookieDelete](#cookiedelete)
    * [CookieNew](#cookienew)
    * [CookieReply](#cookiereply)
    * [CurrentServer](#currentserver)
    * [Deregister](#deregister)
    * [DiscoOrganizations](#discoorganizations)
    * [DiscoServers](#discoservers)
    * [ExpiryTimes](#expirytimes)
    * [FreeString](#freestring)
    * [GetConfig](#getconfig)
    * [InState](#instate)
    * [Register](#register)
    * [RemoveServer](#removeserver)
    * [RenewSession](#renewsession)
    * [ServerList](#serverlist)
    * [SetProfileID](#setprofileid)
    * [SetSecureLocation](#setsecurelocation)
    * [SetState](#setstate)
    * [SetSupportWireguard](#setsupportwireguard)
    * [SetTokenHandler](#settokenhandler)
    * [StartFailover](#startfailover)

# About the API
package main implements the main exported API to be used by other languages

Some notes:

- Errors are returned as JSON c strings. The JSON type is defined in
types/error/error.go Error. Free them using FreeString. Same is the case for
other string types, you should also free them

- Types are converted from the Go representation to C using JSON strings

- Cookies are used for cancellation, just fancy contexts. Create a cookie using
`CookieNew`, pass it to the function that needs one as the first argument. To
cancel the function, call `CookieCancel`, passing in the same cookie as argument

- Cookies must also be freed, by using the CookieDelete function if the cookie
is no longer needed

- The state machine is used to track the state of a client. It is mainly used
for asking for certain data from the client, e.g. asking for profiles and
locations. But a client may also wish to build upon this state machine to build
the whole UI around it. The SetState and InState functions are useful for this


# Functions
## AddServer
Signature:
 ```go
func AddServer(c C.uintptr_t, _type C.int, id *C.char, ni C.int) *C.char
```
AddServer adds a server to the eduvpn-common server list `c` is the cookie
that is used for cancellation. Create a cookie first with CookieNew.
This same cookie is also used for replying to state transitions

`_type` is the type of server that needs to be added. This type is defined
in types/server/server.go Type

`id` is the identifier of the string:

  - In case of secure internet: The organization ID

  - In case of custom server: The base URL

  - In case of institute access: The base URL

`ni` stands for non-interactive. If non-zero, any state transitions will not
be run.

This `ni` flag is useful for preprovisioned servers. For normal usage,
you want to set this to zero (meaning: False)

If the server cannot be added it returns the error as types/error/error.go
Error

Note that the server is removed when an error has occured The following
state callbacks are mandatory to handle:
  - OAUTH_STARTED: This indicates that the OAuth procedure has been started,
    it returns the URL as the data. The client should open the webbrowser
    with this URL and continue the authorization process.

## Cleanup
Signature:
 ```go
func Cleanup(c C.uintptr_t) *C.char
```
Cleanup sends a /disconnect to cleanup the connection

This MUST be called when disconnecting, see
https://docs.eduvpn.org/server/v3/api.html#application-flow `c` is the
Cookie that needs to be passed. Create a new Cookie using `CookieNew`

If it was unsuccessful, it returns an error

## CookieCancel
Signature:
 ```go
func CookieCancel(c C.uintptr_t) *C.char
```
CookieCancel cancels the cookie

This means that functions which take this as first argument, return if
they're still running The error cause is always context.Canceled for that
cancelled function: https://pkg.go.dev/context#pkg-variables

This CookieCancel function can also return an error if cancelling was
unsuccessful

## CookieDelete
Signature:
 ```go
func CookieDelete(c C.uintptr_t) *C.char
```
CookieDelete deletes the cookie by cancelling it and deleting the underlying
cgo handle

This function MUST be called when the cookie that is created using
`CookieNew` is no longer needed

## CookieNew
Signature:
 ```go
func CookieNew() C.uintptr_t
```
CookieNew creates a new cookie and returns it

This value should not be parsed or converted somehow by the client This
value is simply to pass back to the Go library This value has two purposes:

  - Cancel a long running function

  - Send a reply to a state transition (ASK_PROFILE and ASK_LOCATION)

Functions that take a cookie have it as the first argument

## CookieReply
Signature:
 ```go
func CookieReply(c C.uintptr_t, data *C.char) *C.char
```
CookieReply replies to a state transition using the cookie

The data that is sent to the Go library is the second argument of this
function

  - `c` is the Cookie

  - `data` is the data to send, e.g. a profile ID

## CurrentServer
Signature:
 ```go
func CurrentServer() (*C.char, *C.char)
```
CurrentServer gets the current server from eduvpn-common

In eduvpn-common, a server is marked as 'current' if you have gotten a VPN
configuration for it

It returns the server as JSON, defined in types/server/server.go Current

If there is no current server or some other, e.g. there is no current state,
an error is returned with a nil string

## Deregister
Signature:
 ```go
func Deregister() *C.char
```
Deregister cleans up the state for the client.

# If no client is available or deregistering fails, it returns an error

This function SHOULD be called when the application exits such that the
configuration file is saved correctly. Note that saving of the configuration
file also happens in other cases, such as after getting a VPN configuration.
Thus it is often not problematic if this function cannot be called due to a
client crash

## DiscoOrganizations
Signature:
 ```go
func DiscoOrganizations(c C.uintptr_t) (*C.char, *C.char)
```
DiscoOrganizations gets the organizations from discovery, returned as
types/discovery/discovery.go Organizations marshalled as JSON

`c` is the Cookie that needs to be passed. Create a new Cookie using
`CookieNew`

If it was unsuccessful, it returns an error. Note that when the lib was
built in release mode the data is almost always non-nil, even when an error
has occurred This means it has just returned the cached list

## DiscoServers
Signature:
 ```go
func DiscoServers(c C.uintptr_t) (*C.char, *C.char)
```
DiscoServers gets the servers from discovery, returned as
types/discovery/discovery.go Servers marshalled as JSON

`c` is the Cookie that needs to be passed. Create a new Cookie using
`CookieNew`

If it was unsuccessful, it returns an error. Note that when the lib was
built in release mode the data is almost always non-nil, even when an error
has occurred This means it has just returned the cached list

## ExpiryTimes
Signature:
 ```go
func ExpiryTimes() (*C.char, *C.char)
```
ExpiryTimes gets the expiry times for the current server

Expiry times are just fields that represent unix timestamps at which to
do certain events regarding expiry, e.g. when to show the renew button,
when to show expiry notifications

The expiry times structure is defined in types/server/server.go `Expiry` If
some error occurs, it is returned as types/error/error.go Error

## FreeString
Signature:
 ```go
func FreeString(addr *C.char)
```
FreeString frees a string that was allocated by the eduvpn-common Go library

This happens when we return strings, such as errors from the Go lib back to
the client. The client MUST thus ensure that this memory is freed using this
function. Simply pass the pointer to the string in here

## GetConfig
Signature:
 ```go
func GetConfig(c C.uintptr_t, _type C.int, id *C.char, pTCP C.int, startup C.int) (*C.char, *C.char)
```
GetConfig gets a configuration for the server

`c` is the cookie that is used for cancellation. Create a cookie first with
CookieNew, this same cookie is also used for replying to state transitions

`_type` is the type of server that needs to be added. This type is defined
in types/server/server.go Type

`id` is the identifier of the string

  - In case of secure internet: The organization ID
  - In case of custom server: The base URL
  - In case of institute access: The base URL

`pTCP` is if we prefer TCP or not to get the configuration, non-zero means
yes

`startup` is if the client is just starting up, set this to true (non-zero)
if you autoconnect to a server on startup. If this startup value is true
(non-zero) then any authorization or other callacks (profile/location) are
not triggered

After getting a configuration, the FSM moves to the GOT_CONFIG state
The return data is the configuration, marshalled as JSON and defined in
types/server/server.go Configuration

If the config cannot be retrieved it returns an error as
types/error/error.go Error.

The current state callbacks MUST be handled:

### ASK_PROFILE

This asks the client for profile.

This is called when the user/client has not set a profile for this server
before, or the current profile is invalid

When the user has selected a profile, reply with the choice using the
`CookieReply` function and the profile ID e.g. CookieReply(cookie,
"wireguard"). CookieReply can be done in the background as the Go library
waits for a reply

The data for this transition is defined in types/server/server.go
RequiredAskTransition with embedded data Profiles in types/server/server.go.
Note that RequiredTransition contains the cookie to be used for the
CookieReply

So a client would:

- Parse the data to get the cookie and data

- get the cookie

- get the profiles from the data

- show it in the UI and then reply with CookieReply using the choice

### ASK_LOCATION

This asks the client for a location. Note that under normal circumstances,
this callback is not actually called as the home organization for the
secure internet server is set as the current if for some reason, an invalid
location has been configured, the library will ask the client for a new one

When the user has selected a location, reply with the choice using the
`CookieReply` function and the location ID e.g. CookieReply(cookie, "nl")

CookieReply can be done in the background as the Go library waits for a
reply The data for this transition is defined in types/server/server.go
RequiredAskTransition with embedded data a list of strings ([]string)

Note that RequiredTransition contains the cookie to be used for the
CookieReply,

So a client would:

  - Parse the data to get the cookie and data

  - get the cookie

  - get the list of locations from the data

  - show it in the UI and then reply with CookieReply using the choice

### OAUTH_STARTED

This indicates that the OAuth procedure has been started, it returns the URL
as the data.

The client should open the webbrowser with this URL and continue the
authorization process. This is only called if authorization needs to be
retriggered

## InState
Signature:
 ```go
func InState(fsmState C.int) (C.int, *C.char)
```
InState checks if the FSM is in `fsmState`

## Register
Signature:
 ```go
func Register(
```
	name *C.char,
	version *C.char,
	configDirectory *C.char,
	cb C.StateCB,
	debug C.int,
) *C.char
Register creates a new client and also registers the FSM to go to the
initial state

`Name` is the name of the client, must be a valid client ID

`Version` is the version of the client. This version field is used for the
user agent in all HTTP requests

`cb` is the state callback. It takes three arguments: The old state, the new
state and the data for the state as JSON

  - Note that the states are defined in client/fsm.go, e.g. NO_SERVER (in
    Go: StateNoServer), ASK_PROFILE (in Go: StateAskProfile)

  - This callback returns non-zero if the state transition is handled.
    This is used to check if the client handles the needed transitions

debug, if non-zero, enables debugging mode for the library, this means:

  - Log everything in debug mode, so you can get more detail of what is
    going on

  - Write the state graph to a file in the configDirectory. This can be used
    to create a FSM png file with mermaid https://mermaid.js.org/

After registering, the FSM is initialized and the state transition NO_SERVER
should have been completed If some error occurs during registering, it is
returned as a types/error/error.go Error

## RemoveServer
Signature:
 ```go
func RemoveServer(_type C.int, id *C.char) *C.char
```
RemoveServer removes a server from the eduvpn-common server list

`_type` is the type of server that needs to be added. This type is defined
in types/server/server.go Type

`id` is the identifier of the string:

  - In case of secure internet: The organization ID

  - In case of custom server: The base URL

  - In case of institute access: The base URL

If the server cannot be removed it returns the error types/error/error.go
Error Note that the server is not removed when an error has occured

## RenewSession
Signature:
 ```go
func RenewSession(c C.uintptr_t) *C.char
```
RenewSession renews the session of the VPN

This essentially means that the OAuth tokens are deleted. And it also
possibly re-runs every state callback you need when getting a config.
So least you MUST handle the OAuth started transition

It returns an error if unsuccessful

## ServerList
Signature:
 ```go
func ServerList() (*C.char, *C.char)
```
ServerList gets the list of servers that are currently added

This is NOT the discovery list, but the servers that have previously been
added with `AddServer`

It returns the server list as a JSON string defined in
types/server/server.go List

If the server list cannot be retrieved it returns a nil string and an error

## SetProfileID
Signature:
 ```go
func SetProfileID(data *C.char) *C.char
```
SetProfileID sets the profile ID of the current serrver

This MUST only be called if the user/client wishes to manually set a profile
instead of the common lib asking for one using a transition

`Data` is the profile ID

It returns an error if unsuccessful

## SetSecureLocation
Signature:
 ```go
func SetSecureLocation(c C.uintptr_t, data *C.char) *C.char
```
SetSecureLocation sets the location for the secure internet server if it
exists

This MUST only be called if the user/client wishes to manually set a
location instead of the common lib asking for one using a transition

Because this does network requests to initialize the location, there is a
cookie again :)

`c` is the Cookie that needs to be passed. To create a cookie, first call
`CookieNew` `Data` is the location ID

It returns an error if unsuccessful

## SetState
Signature:
 ```go
func SetState(fsmState C.int) *C.char
```
SetState sets the state of the statemachine

Note: this transitions the FSM into the new state without passing any data
to it

## SetSupportWireguard
Signature:
 ```go
func SetSupportWireguard(support C.int) *C.char
```
SetSupportWireguard enables or disables WireGuard for the client

By default WireGuard support is enabled To disable it you can pass a 0 int
to this

`support` thus indicates whether or not to enable WireGuard An error is
returned if this is not possible

## SetTokenHandler
Signature:
 ```go
func SetTokenHandler(getter C.TokenGetter, setter C.TokenSetter) *C.char
```
SetTokenHandler sets the token getters and token setters for OAuth

Because the data that is saved does not contain OAuth tokens for server,
the common lib asks and sets the tokens using these callback functions.
The client can thus pass callbacks to this function so that the tokens can
be securely stored in a keyring

Client must pass two arguments to these functions

  - getter is the void function that gets tokens from the client. It takes
    three arguments:

  - The `server` for which to get the tokens for, marshalled as JSON and
    defined in types/server/server.go `Current`

  - The `output` buffer

  - The `length` of the output buffer. This 'output buffer' must contain the
    tokens, marshalled as JSON that is defined in types/server/server.go
    `Tokens`

setter is the void function that sets tokens. It takes two arguments:

  - The `server` for which to get the tokens for, marshalled as JSON and
    defined in types/server/server.go `Current`

  - The `tokens`, defined in types/server/server.go `Tokens` marshalled as
    JSON

It returns an error when the tokens cannot be set

## StartFailover
Signature:
 ```go
func StartFailover(c C.uintptr_t, gateway *C.char, mtu C.int, readRxBytes C.ReadRxBytes) (C.int, *C.char)
```
StartFailover starts the 'failover' procedure in eduvpn-common

Failover has one primary goal: check if the VPN can reach the gateway.
This can be used to check whether or not the client needs to 'failover' to
prefer TCP (if currently using UDP). Which is useful to go from a broken
WireGuard connection to OpenVPN over TCP

  - `c` is the cookie that is passed for cancellation. To create a cookie,
    use the `CookieNew` function
  - `gateway` is the gateway IP of the VPN
  - `readRxBytes` is a function that returns the current rx bytes of the VPN
    interface, this should return a `long long int` in c

It returns a boolean whether or not the common lib has determined that it
cannot reach the gateway. Non-zero=dropped, zero=not dropped.

It also returns an error, if it fails to indicate if it has dropped or not.
In this case, dropped is also set to zero

