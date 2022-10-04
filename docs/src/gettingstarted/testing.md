# Testing
The Go library right now has tests defined for server interaction and signature verification tests.

To run the test suite, issue the following command in a shell

```bash
make test-go
```

Note that this runs the tests without any server interaction (so for now only the signature verification tests). To run the tests with an eduVPN server you need to specify environment variables:

```bash
SERVER_URI="eduvpn.example.com" PORTAL_USER="example" PORTAL_PASS="example" make test-go
```

This needs [python3-selenium](https://selenium-python.readthedocs.io/) and [geckodriver](https://github.com/mozilla/geckodriver/releases) (extract and put in your `$PATH`).

If you have [Docker](https://www.docker.com/get-started/) installed and [Docker-compose](https://docs.docker.com/compose/install/) you can use a convenient helper script which starts up two containers
- An eduVPN server for testing
- A Go container that builds and runs the test-suite

```bash
PORTAL_USER="example" PORTAL_PASS="example" ./ci/startcompose.sh
```
Note that this helper script also assumes you have the [OpenSSL](https://www.openssl.org/) command line tool installed. This is used to install the self-signed certificates for testing.

This script is also used in the continuous integration, so we recommend to run this before you submit any changes.

There are other environment variables that can be used:

- `OAUTH_EXPIRED_TTL`: Use this for a server which has a low OAuth access token expiry time, e.g. 10 seconds. You would then set this variable to `"10"` so that a test is ran which waits for 10 seconds for the OAuth tokens to expire
## Testing the wrappers
To test the wrappers, issue the following command in a shell (you will need dependencies for all wrappers if you do this[^1]):

```bash
make test-wrappers
```

Specify `-j` to execute tests in parallel. You can specify specific wrappers to test by appending
e.g. `WRAPPERS="csharp php"`.

## Test everything
To test all the code at once, issue the following command:
```bash
make test
```

This accepts the same environment variables as we have explained before.

[^1]: For now, this is only the Python wrapper as the other wrappers do not implement the newest API just yet.
