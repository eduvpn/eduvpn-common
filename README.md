# EduVPN shared library

This repository contains a Go library with functions that all EduVPN clients can use. The goal is to let EduVPN clients
link against this library and gradually merge more common logic between EduVPN clients into this repository.

[cgo](https://pkg.go.dev/cmd/cgo) is used to build the Go library into a shared dynamic library. Wrappers will be
written using some FFI framework for each language used in EduVPN clients to easily interface with the library.

## Functionality

Currently, only verification of signatures on files from `disco.eduvpn.org` is supported. For now, these files have to
be downloaded by the caller.

⚠️ The caller has to extract the timestamp from the file (JSON `"v"` tag), save it, and pass it to the Verify function
the next time that they call it. This prevents a rollback of a previous file. This functionality may be integrated into
the library in the future.

## Requirements

To run the Go tests, you will need [Go](https://go.dev/doc/install) 1.15 or later (add it to your `PATH`). To build the
shared library, you will additionally need to install gcc. If you want to use the Makefile scripts you will need GNU
make (not bsd make).

On Windows, you can install gcc and make (or even Go) via MinGW or Cygwin or use WSL. For MinGW:

1. [Install MinGW](https://www.msys2.org/#installation) (you don't need to install any extra packages yet) and open some
   MSYS2 terminal (e.g. from the start menu or one of the installed binaries)
2. Install the [`make`](https://packages.msys2.org/package/make?repo=msys) package (`pacman -S make`) (or
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

Take a look at `wrappers/<lang>/README.md` for extra instructions for each wrapper.

## Build & test

Build shared library for current platform:

```shell
make
```

Build shared library for specified OS & architecture (example):

```shell
make GOOS=windows GOARCH=386
```

To list all platforms supported by cgo, run `go tool dist list`.

Results will be output in `exports/lib/`.

Usually you will need to specify the compiler when cross compiling, for example:

```shell
make GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc
```

For example, you can cross compile for Windows from Linux using [MinGW-w64](https://www.mingw-w64.org/downloads/).

Test Go code:

```shell
make test-go
```

Test wrappers (you will need compilers for all wrappers if you do this):

```shell
make test-wrappers
```

Specify `-j` to execute tests in parallel. You can specify specific wrappers to test by appending
e.g. `WRAPPERS="csharp php"`.

Test both:

```shell
make test
```

Clean built libraries and wrapper builds:

```shell
make -j clean
```

Usually you won't need to do this, as changes in the library should automatically be incorporated in wrappers.
Specify `CLEAN_ALL=1` to also remove downloaded dependencies for some wrappers. You can clean individual wrappers by
executing clean in their directories, or specify `WRAPPERS=...`.

Take a look at `wrappers/<lang>/README.md` for descriptions per wrapper.

## Structure

- `verify.go`: main API
- `verify_test.go` and `test_data/`: tests for API
- `exports/`: C API interface
- `exports/lib/`: built libraries per architecture per OS
- `wrappers/`: wrappers per language
