# Swift wrapper

## Requirements

You will need to install the [Swift SDK](https://www.swift.org/getting-started), which includes the `swift` tool.

## Build & test

Build `EduVpnCommon` using shared Go library for current platform:

```shell
make
```

Build `EduVpnCommon` using shared Go library for specified platform, e.g.:

```shell
make GOOS=linux GOARCH=amd64
```

On Windows, you will also need to generate a .lib for the .dll.

Test:

```shell
make test
```
