# C# wrapper

## Requirements

You will need to install the [.NET SDK](https://dotnet.microsoft.com/download), which includes the `dotnet` tool. The
wrapper targets .NET Standard 2.0, so which means that at least .NET Core 2.0 is required (.NET 5+ is also fine). For
the tests, .NET 5 or newer is required.

## Build & test

First build the shared Go library. Next:

Build `EduVpnCommon`:

```shell
make
```

Build as nupkg, including eduvpn_verify library:

```shell
make pack
```

The wrapper targets .NET Standard 2.0, which means that it can be referenced by projects using either .NET Framework
4.6.1+, .NET Core 2.0+, or .NET 5+.

Currently, directly referencing the project may not work (with `System.BadImageFormatException`) if you have multiple
dynamic libraries compiled in the `exports` folder. If you instead add the `.nupkg`, e.g. with one of the
methods [here](https://stackoverflow.com/q/43400069) or [here](https://stackoverflow.com/q/10240029), it automatically
copies the correct library.

Test:

```shell
make test
```
