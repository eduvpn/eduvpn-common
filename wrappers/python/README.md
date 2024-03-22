# Python wrapper

## Requirements

Python 3.6+ is assumed, but it may work with older versions. To build, `setuptools`, `wheel` and `build` are required.

## Building

First build the shared Go library by following the instructions in the root directory of this Repo.

Then, to build the wheel use:

```shell
make pack
```

To install the wheel, run:

```shell
pip install dist/eduvpncommon-[version]-py3-none-[platform].whl
```

## Running tests
```shell
make test
```
