# Python wrapper

## Requirements

Python 3.6+ is assumed, but it may work with older versions. To build, `setuptools` and `wheel` are required.

## Build & test

First build the shared Go library. Next:

Build wheel using library for current platform:

```shell
make pack
```

(This does not build the shared Go library.)

Build wheel using library for specified platform (passed to setuptools `--plat-name`,
see [`get_build_platform`](https://setuptools.pypa.io/en/latest/pkg_resources.html?highlight=get_build_platform#platform-utilities)
for more):

```shell
make pack PLAT_NAME=win32
```

To install the wheel, run:

```shell
pip install dist/eduvpncommon-[version]-py3-none-[platform].whl
```

You could also reference the discovery module directly and copy the library for the platform to the `eduvpncommon/lib/`
folder.

If you do not build this as part of the full repository, specify `EXPORTS_PATH="path/to/exports-folder"` when calling
make. This folder must contain `platform.mk` and the `lib/` folder with built libraries.

Test:

```shell
make test
```
