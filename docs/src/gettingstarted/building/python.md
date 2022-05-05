# Python Wrapper

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
