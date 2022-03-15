---
layout: default
title: Home
nav_order: 1
description: "eduVPN-common documentation."
permalink: /
---

## eduVPN-common documentation

This is the documentation for eduVPN-common, a shared GO library to be used as a common codebase between eduVPN clients. This library is a WIP and is not production ready.

### Dependencies
#### Linux
To build the GO shared library using Linux you need the following dependencies:

- [Go](https://go.dev/doc/install) 1.15 or later
- [Gcc](https://gcc.gnu.org/)
- [GNU Make](https://www.gnu.org/software/make/)
- Dependencies for each wrapper you are interested in

#### Windows
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

### Building
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

### Testing

To tesst wrappers, issue the following command in a shell (you will need compilers for all wrappers if you do this):

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
