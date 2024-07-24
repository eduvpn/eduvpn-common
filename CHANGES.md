# 2.1.0 (2024-07-25)
* Discovery:
  - Fetch on startup in the background with a function "DiscoveryStartup"
  - Remove organization 4 hour expiry, fetch organization list when the client needs to authorize again
  - Fix missing JSON fields not having empty values but having values for previous entries due to a memory re-use bug (#55). This can be noticed in e.g. the keyword list
  - Implement conditional requests using `If-Modified-Since` request header and `Last-Modified` response header and checking on HTTP 304
* FSM:
  - Allow AskProfile selection after authorizing
  - Go to previous state on renew error
* Proxyguard integration:
  - Fix race condition warning
* User agent:
  - Override ProxyGuard user agent
  - Override eduoauth-go user agent
* Deps:
  - Update to latest versions

# 2.0.2 (2024-06-25)
* Client: More frequent state file saving which helps forceful closes of a client
* Config: Implement atomic file writes

# 2.0.1 (2024-06-06)
* Discovery: Updates regarding the internal handling of the discovery organization list cache
* Translations: Update from weblate

# 2.0.0 (2024-06-04)
* Minimise exposed discovery to the client and add search by giving a second argument to DiscoServers or DiscoOrganizations with the search query:
  - organization list globally changes:
    * remove `v` field
    * remove `go_timestamp` field
  - each organization changes:
    * remove `authentication_url_template` field
    * remove `keyword_list` field
    * remove `public_key_list` field
    * remove `support_contact` field
  - server list globally changes:
    * remove `v` field
    * remove `go_timestamp` field
  - each server changes:
    * remove `secure_internet_home` field
    * remove `keyword_list` field
* Python wrapper:
  - Add setup.py & setup.cfg for backwards compatibility instead of completely relying on pyproject.toml
* API:
  - Add a ton of tests by mocking the server API
* Makefile:
  - Add a coverage target
* Server:
  - Replace the non-interactive AddServer flag with an oauth start time flag,
    if non-nil the server is added non-interactively and the OAuth start time is stored
* Client FSM:
  - Remove graph image generation using Mermaid as that is too much code in core, this will be implemented using an external script
* Example CLI:
  - Fix profile/location selection
* Translations:
  - Update from Weblate
* Deps:
  - Update to eduoauth-go 1.0.0

# 1.99.2 (2024-04-25)
* Expose default gateway in profile settings too. For clients, use the default gateway set on the config object, this is maybe only useful for suggesting some profiles in the client profile chooser UI
* Add a server internally before authorizing and remove it again if authorization has failed. This makes sure the internal state is always up-to-date with what is happening. This also allows us to move to the main state when authorization is done as previously it could be the case where authorization was done but the server was not added yet
* Fix previous state not being set correctly when getting a config and an error happens
* Make WireGuard support mandatory
* Cache secure internet profile choice per location
* Update go dependencies: eduoauth-go logging changes
* Cancel ProxyGuard in Cleanup function if it was started using the common functionality
* FSM Changes: Allow to go to disconnected from OAuthStarted and GettingConfig
* Refactor makefile & building for Go and Python code

# 1.99.1 (2024-03-11)
* Disable type annotation for global eduVPN class as it gave a `SyntaxError` on some Python versions. See https://bugs.python.org/issue34939

# 1.99.0 (2024-03-07)
* OAuth:
    - Move to github.com/jwijenbergh/eduoauth-go
* WireGuard:
    - Support WireGuard over TCP using https://codeberg.org/eduVPN/proxyguard
* Dependencies:
    - Remove github.com/go-errors/errors
* State:
    - Create a new state file (v2), but automatically convert from version 1 to 2
    - Remove a ton of caching in the version 2 state file
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
* API:
    - add support for DNS search domains
    - add support for VPN proto transport list
* Server List:
    - Implement `delisted` servers

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
