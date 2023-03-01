# Building the Go library
To build the Go library, you need the dependencies for your system installed. We will go over the needed dependencies for Linux and Windows. Afterwards, we explain the basic commands to build the library.

## Dependencies
### Linux
To build the Go shared library using Linux you need the following dependencies:

- [Go](https://go.dev/doc/install) 1.18 or later
- [Gcc](https://gcc.gnu.org/)
- [GNU Make](https://www.gnu.org/software/make/)
- Dependencies for each wrapper you are interested in (read next sections)

### Windows
On Windows, you can install gcc and make (or even Go) via MinGW or Cygwin or use WSL. For MinGW:

1. [Install MinGW](https://www.msys2.org/#installation) (you don't need to install any extra packages yet) and open some
   MSYS2 terminal (e.g. from the start menu or one of the installed binaries)
2. Install the [`make`](https://packages.msys2.org/package/make?repo=msys) package (or
   e.g. [`mingw-w64-x86_64-make`](https://packages.msys2.org/package/mingw-w64-x86_64-make?repo=mingw64) and
   use `mingw32-make` in the command line)
3. To compile for x86_64:
    1. Install the [`mingw-w64-x86_64-gcc`](https://packages.msys2.org/package/mingw-w64-x86_64-gcc?repo=mingw64)
       package
    2. Open the MinGW 64-bit console, via the start menu, or in your current
       terminal: `path/to/msys64/msys2_shell.cmd -mingw64 -defterm -no-start -use-full-path`
    3. Run the make commands in the project directory
4. To compile for x86 (32-bit):
    1. Install the [`mingw-w64-i686-gcc`](https://packages.msys2.org/package/mingw-w64-i686-gcc?repo=mingw32) package
    2. Open the MinGW 32-bit console, via the start menu, or in your current
       terminal: `path/to/msys64/msys2_shell.cmd -mingw32 -defterm -no-start -use-full-path`
    3. Run the make commands in the project directory

## Commands
Before we can begin building the wrapper code, we need to build the Go code as a shared library. This section will tell you how to do so.

To build the shared library for the current platform issue the following command in the root directory:

```bash
make
```

You can also build the shared library for a specified OS & architecture (example):

```bash
make GOOS=windows GOARCH=386
```

We use cgo to build a shared library, to list all platform supported by cgo issue `go tool dist list`.

The shared library will be output in `exports/lib/`.

For cross compiling, you usually need to specify the compiler, for example:

```bash
make GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc
```

For example, you can cross compile for Windows from Linux using [MinGW-w64](https://www.mingw-w64.org/downloads/).

This shared library gets loaded by the different wrappers. To build the actual wrapper code, you need other build commands. This will be explained now.

### Cleaning
To clean build the library and wrapper, issue the following command in the root directory:

```bash
make -j clean
```

Usually you won't need to do this, as changes in the library should automatically be incorporated in wrappers.
Specify `CLEAN_ALL=1` to also remove downloaded dependencies for some wrappers. You can clean individual wrappers by
executing clean in their directories, or specify `WRAPPERS=...`.

## Note on releases
Releases are build with the go tag "release" (add flag "-tags=release") to bundle the discovery JSON files and embed them in the shared library. See the [make_release](https://github.com/eduvpn/eduvpn-common/blob/main/make_release.sh) script on how we bundle the files. A full command without the Makefile to build this library is:

```bash
go build -o lib/libeduvpn_common-${VERSION}.so -tags=release -buildmode=c-shared ./exports
```
