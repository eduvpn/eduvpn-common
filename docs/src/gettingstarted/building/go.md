# Building the Go library
To build the Go library, you need the dependencies for your system installed. We will go over the needed dependencies for Linux. Afterwards, we explain the basic commands to build the library.

## Dependencies
### Linux
To build the Go shared library using Linux you need the following dependencies:

- [Go](https://go.dev/doc/install) 1.18 or later
- [Gcc](https://gcc.gnu.org/)
- [GNU Make](https://www.gnu.org/software/make/)
- Dependencies for the Python wrapper if you want to build that as well

## Commands
Before we can begin building the wrapper code, we need to build the Go code as a shared library. This section will tell you how to do so.

To build the shared library for the current platform issue the following command in the root directory:

```bash
make
```

The shared library will be output in `lib/`.

### Cleaning
To clean build the library and wrapper, issue the following command in the root directory:

```bash
make clean
```

## Note on releases
Releases are build with the go tag "release" (add flag "-tags=release") to bundle the discovery JSON files and embed them in the shared library. See the [make_release](https://codeberg.org/eduVPN/eduvpn-common/src/branch/main/make_release.sh) script on how we bundle the files. A full command without the Makefile to build this library is:

```bash
go build -o lib/libeduvpn_common-${VERSION}.so -tags=release -buildmode=c-shared ./exports
```
