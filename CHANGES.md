# 1.2.1 (2024-05-24)
* Python wrapper:
    - Do not add type annotation for global object
    - Do not use \* in docstring to avoid SyntaxWarning for Python 3.12

# 1.2.0 (2023-12-08)
* OAuth:
    - Disable all ISS checks
* API:
    - Only validate that the scheme is HTTPS in well-known. Hostname checking should be handled by ISS properly in the future
* Renew:
    - Base showing renew button time on 30 minutes since authorization start time correctly
    - Add docker files for building pip packages

# 1.1.99.0 (1.2 pre-release) (2023-10-25)
* Version:
    - Make it easier to update
* Makefile:
    - Remove unused packaging targets
* CI/Docker (cherry-picked from main branch):
    - Use https://codeberg.org/eduvpn/deploy instead of https://codeberg.org/eduvpn/documentation for the deployment scripts
    - Update Fedora 36 to 38
* go.mod/go.sum:
    - Update dependencies
* Release scripts:
    - Add `prepare_release.sh` script that formats the code and changes version numbers automatically
    - Change `make_release.sh` to automatically add artifacts
    - Support development GPG key for pre-releases

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
