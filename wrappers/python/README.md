# Python wrapper

## Requirements

Python 3.6+ is assumed, but it may work with older versions.

TODO Build

## Build & test

First build the shared Go library. Next:

Build wheel using library for current platform:

```shell
make
```

Build wheel using library for specified platform (passed to setuptools `--plat-name`):

```shell
make PLAT_NAME=win32
```

To install the wheel, run:

```shell
pip install dist/eduvpncommon-[version]-py3-none-[platform].whl
```

You could also reference the discovery module directly and copy the library for the platform to the `eduvpncommon/lib`
folder.

Test:

```shell
make test
```
