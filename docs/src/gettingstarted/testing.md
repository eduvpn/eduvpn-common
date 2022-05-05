# Testing
To test the Go code, issue the following command in a shell

```bash
make test-go
```

Note that this runs the tests without any server interaction. To run the tests with an eduVPN server you need to specify environment variables:

```bash
SERVER_URI="eduvpn.example.com" PORTAL_USER="example" PORTAL_PASS="example" make test-go
```

If you have [Docker](https://www.docker.com/get-started/) installed and [Docker-compose](https://docs.docker.com/compose/install/) you can use a convenient helper script which starts up two containers
- An EduVPN Server for testing
- A Go container that builds and runs the test-suite

```bash
PORTAL_USER="example" PORTAL_PASS="example" ./ci/startcompose.sh
```
Note that this helper script also assumes you have the `openssl` command line tool installed. This is used to install the self-signed certificates for testing.

This script is also used in the continuous integration, so we recommend to run this before you submit any changes.
## Testing the wrappers
To test the wrappers, issue the following command in a shell (you will need compilers for all wrappers if you do this):

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
