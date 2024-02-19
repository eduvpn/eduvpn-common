# Unreleased
* OAuth:
    - Move to github.com/jwijenbergh/eduoauth-go
* WireGuard:
    - Support WireGuard over TCP using https://codeberg.org/eduVPN/proxyguard
* Dependencies:
    - Remove github.com/go-errors/errors
* State:
    - Create a new state file (v2), but automatically convert from version 1 to 2
* Data transmission:
    - Move from Go->C types->language X types to using: Go struct -> JSON as a c string -> Language unmarshalls JSON. This eliminates a lot of code and makes it easier for clients to Go and the clients to convert data.
	- The data that clients receive is handled in the `types` folder
	- Rewrite the Python wrapper to use this new API but only return the JSON string. The Linux client further handles the processing of the data. This makes the wrapper nice and small
* Client package:
    - Remove server.go file and merge into the main implementation file (client.go)
* Internal server package:
    - Split into subpackages
* Failover:
    - Support Windows by using an ip4:icmp ping instead of udp4
* Errors:
    - Translate errors that are returned to clients using golang.org/x/text and use Weblate
    - Split into "internal" errors and actual errors. Internal errors are errors that *should not* happen and will thus also not be translated. These internal errors are mostly due to a client fault, e.g. trying to get discovery servers when the client is Let's Connect!
* Cancellation:
    - Support contexts in the Go API such that almost any action can be cancelled, e.g. HTTP requests.
    - Support these contexts in the exported API by creating so-called "cookies". The way it works is that clients create a cookie and then pass it to a function. When the client was to cancel any function that uses this cookie it calls "CookieCancel". These same cookies are also used as identifiers to reply to state transitions, e.g. "here is the profile I have chosen" or "here is the secure internet location I want to choose".
* FSM:
    - Properly restore the previous state when an error occurs instead of almost always going back to `NoServer`
    - Simplified to use less states
* CI + Docker:
    - Use https://codeberg.org/eduvpn/deploy instead of https://codeberg.org/eduvpn/documentation for the deployment scripts
* Docs:
    - Autogenerate exports docs using genexportsdoc.py
    - Rewrite a large portion of the API section
    - Support mermaid graphs using mdbook-mermaid

# 1.1.2 (2023-09-01)
* Server:
    - Update endpoints more frequently
    - Update endpoints differently for secure internet: For the "home server" and the "current location" separately
* Python:
    - Change setup.py lib copying to fix pip building with manylinux
* Deps:
    - Update go.mod/go.sum

# 1.1.1 (2023-08-29)
* Server:
    - Update OAuth endpoints when endpoints are refreshed from .well-known/vpn-user-portal

# 1.1.0 (2023-04-18)
* Client:
    - Implement a toker updater callback to notify client of any token updates
    - Make sure we go back in the FSM when we have an error with setting a custom server
    - Log current secure internet server we're getting a config for
* Server:
    - Fixed error wrapping when a server is not a custom server, it tried to wrap an empty error
* OAuth
    - Set previous refresh token if new refresh token is empty after refreshing. This is needed for 2.x servers
* Exports
    - Safeguard against nil servers and organizations. This should not happen in production due to it always being non-nil. But if clients do not check correctly if they're Let's Connect! or when building in non-release mode, this can be a problem. Now we return an error properly
* Python:
    - Make profiles optional in the server types
* Misc:
    - Use callStr in verify test code to get rid of linting errors
    - Fix line numbers not showing up in linter workflows

# 1.0.0 (2023-03-01)
* Client:
    - Modify user agent to be equal to upstream ClientID server values in https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/VpnClientDb.php
	- Make sure we do not constrain the version field too much in the user agent by allowing 20 characters
	- Fix unfriendly error message when a secure internet location cannot be loaded
* Docs:
    - Add section for release building
* Release:
    - Add a script to release tarballs and sign them

# 0.99.0 (2023-03-01)
* Discovery:
    - Bundle on release using embed
    - Cache in the JSON
* Errors:
    - Remove error levels for now
    - Improve the context
    - Use `errors.New` when we can
* HTTP:
	- Add tests
    - Implement some utility functions for paths
    - Set a user agent
* OAuth:
    - Make ISS required
    - Only handle the token authorization callback request once
    - Add logging for token flow
* Server:
    - Add profile tests
    - Validate endpoints to have the same scheme and hostname
* General:
    - Update dependencies
    - Use one logger instance

# 0.3.0 (2023-02-01)
* Discovery:
    - Add tests with a local TLS server
    - Implement expiry/caching closer to spec
* CI:
    - Run without docker to speed up testing
    - Update docker script to support podman as well. Can be used to run the tests locally
* Python: Use Let's Connect! for the tests to disable discovery
* HTTP:
    - Always enforce HTTPS scheme
    - Limit the maximum data read from the server to 16 MB
* OpenVPN: add script-security 0 to the config to, by default, disallow arbitrary scripts from being run depending on the OpenVPN implementation. This can be overridden by a client
* CLI: Validate OAuth URL scheme and do not open the browser with xdg-open
* Client
    - validate ClientID
    - Separate cleanup from disconnect function
* Failover: Return early if we get a Pong within ping interval seconds

# 0.2.0 (2021-12-23)
* Initial release
