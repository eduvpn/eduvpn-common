#!/usr/bin/env python3

import os
import pathlib
import shutil
import typing
from collections import defaultdict

import sys
from setuptools import setup
from wheel.bdist_wheel import bdist_wheel as _bdist_wheel


def getlibpath(plat_name: str) -> typing.Union[str, None]:
    _plat_map = defaultdict(lambda: plat_name, {
        "win32": "win-x86",
    })

    plat_split = _plat_map[plat_name].split("-", 1)
    if len(plat_split) != 2:
        return None
    plat_os, plat_arch = plat_split

    _os_map = defaultdict(lambda: plat_os, {
        "win": "windows",
    })
    _lib_prefixes = defaultdict(lambda: "lib", {
        "windows": "",
    })
    _lib_suffixes = defaultdict(lambda: ".so", {
        "windows": ".dll",
        "darwin": ".dylib",
    })
    _arch_map = defaultdict(lambda: plat_arch, {
        "aarch64_be": "arm64",
        "aarch64": "arm64",
        "armv8b": "arm64",
        "armv8l": "arm64",
        "x86": "386",
        "x86pc": "386",
        "i86pc": "386",
        "i386": "386",
        "i686": "386",
        "x86_64": "amd64",
        "i686-64": "amd64",
    })

    processed_os = _os_map[plat_os]
    return f"{processed_os}/{_arch_map[plat_arch]}/" \
           f"{_lib_prefixes[processed_os]}eduvpn_verify{_lib_suffixes[processed_os]}"


# You would say there would be a better way to do all of this, but I couldn't find it

class bdist_wheel(_bdist_wheel):
    def run(self):
        self.plat_name_supplied = True  # Force use platform

        libpath = getlibpath(self.plat_name)
        if not libpath:
            print(f"Unknown platform: {self.plat_name}")
            sys.exit(1)

        print(f"Building wheel for platform {self.plat_name}")

        shutil.copy2(f"../../exports/{libpath}", "eduvpncommon/lib/")
        _bdist_wheel.run(self)
        os.remove(f"eduvpncommon/lib/{pathlib.Path(libpath).name}")


setup(
    name="eduvpncommon",
    version="0.1.0",
    packages=["eduvpncommon"],
    python_requires=">=3.6",
    package_data={"eduvpncommon": ["lib/*eduvpn_verify*"]},
    cmdclass={"bdist_wheel": bdist_wheel},
)
