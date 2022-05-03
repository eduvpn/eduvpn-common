# Building
To build the Go library, you need the dependencies for your system installed. We will go over the needed dependencies for Linux and Windows. Afterwards, we explain the basic commands to build the library.
## Linux
To build the GO shared library using Linux you need the following dependencies:

- [Go](https://go.dev/doc/install) 1.15 or later
- [Gcc](https://gcc.gnu.org/)
- [GNU Make](https://www.gnu.org/software/make/)
- Dependencies for each wrapper you are interested in

## Windows
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

## Building Go shared library
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

This shared library gets loaded by the different wrappers. To build the actual wrapper code, you need other build commands. This will be explained now

### Python

To build the python wrapper issue the following command (in the root directory of the eduvpn-common project):

```bash
make -C wrappers/python
```

This uses the makefile in `wrappers/python/Makefile` to build the python file into a wheel placed in `wrappers/python/dist/eduvpncommon-[version]-py3-none-[platform].whl`. Where version is the version of the library and platform is your current platform. Like Go you can also build for a specific platform:

```bash
make PLAT_NAME=win32
```

The wheel can be installed with `pip`:

```bash
pip install ./wrappers/python/dist/eduvpncommon-[version]-py3-none-[platform].whl
```

### Cleaning
Clean built libraries and wrapper builds:

```bash
make -j clean
```

Usually you won't need to do this, as changes in the library should automatically be incorporated in wrappers.
Specify `CLEAN_ALL=1` to also remove downloaded dependencies for some wrappers. You can clean individual wrappers by
executing clean in their directories, or specify `WRAPPERS=...`.

### Example: commands to build for Python
This section gives an example on how to build and install the library from scratch (assuming you have all the dependencies)

1. Clone the library
```bash
git clone https://github.com/jwijenbergh/eduvpn-common
```

2. Go to the library directory
```bash
cd eduvpn-common
```

3. Build the go library
```bash
make
```

4. Build the python wrapper
```bash
make -C wrappers/python
```

5. Install the wheel using pip
```bash
pip install wrappers/python/dist/eduvpncommon-0.1.0-py3-none-linux_x86_64.whl
```
Note that the name of your wheel changes on the platform and version.

