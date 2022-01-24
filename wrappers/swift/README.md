# Swift wrapper

## Requirements

You will need to install the [Swift SDK](https://www.swift.org/getting-started), which includes the `swift` tool. This
project does not require Xcode as it uses the Swift Package Manager.

## Build & test

Build `EduVpnCommon` using shared Go library for current platform:

```shell
make
```

Build `EduVpnCommon` using shared Go library for specified platform, e.g.:

```shell
make GOOS=linux GOARCH=amd64
```

When using this library, you will need to make sure that the linker can find the shared Go library.

<small>On Windows, you will also need to generate a .lib import library for the .dll. You can
use `exports/generate_lib.ps1`
for this, passing in the path to the DLL file. Execute this from a Visual Studio Developer shell before building the
Swift project. Alternatively, you could use `objdump` and `llvm-dlltool`. You only need to update this if the list of
exported symbols changes.</small>

If you just want to copy over the C header file to the right directory for the modulemap in `CEduVpnCommon`, run:

```shell
make install-header
```

If you do not build this as part of the full repository, specify `EXPORTS_PATH="path/to/exports-folder"` when calling
make. This folder must contain `platform.mk` and the `lib/` folder with built libraries and headers.

Test:

```shell
make test
```
