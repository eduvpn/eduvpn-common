# C# wrapper

## Requirements

You will need to install the [.NET SDK](https://dotnet.microsoft.com/download), which includes the `dotnet` tool. The
wrapper targets .NET Standard 2.0, which means that at least .NET Core 2.0 is required (.NET 5+ is also fine). For the
tests, .NET 5 is required.

## Build & test

First build the shared Go library. Next:

Build `EduVpnCommon` (Release):

```shell
make
```

Build as nupkg, including shared Go library for all platforms built in `exports/lib/`:

```shell
make pack
```

If you do not build this as part of the full repository, specify `EXPORTS_PATH="" EXPORTS_LIB_PATH="path/to/lib-folder"`
when calling make.

The wrapper targets .NET Standard 2.0, which means that it can be referenced by projects using either .NET Framework
4.6.1+, .NET Core 2.0+, or .NET 5+.

Currently, directly referencing the project may not work (with `System.BadImageFormatException`) if you have multiple
dynamic libraries compiled in the `exports/lib/` folder. If you instead add the `.nupkg`, e.g. with one of the
methods [here](https://stackoverflow.com/q/43400069) or [here](https://stackoverflow.com/q/10240029), it automatically
copies the correct library.

This also means that tests may fail if you have multiple dynamic libraries compiled!

Test:

```shell
make test
```
