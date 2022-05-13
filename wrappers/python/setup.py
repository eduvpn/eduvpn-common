#!/usr/bin/env python3

import os
import shutil
import sys
import typing
from collections import defaultdict

from setuptools import setup
from wheel.bdist_wheel import bdist_wheel as _bdist_wheel

_libname = "eduvpn_common"


def getlibpath(plat_name: str) -> typing.Union[str, None]:
    """Get library path for plat_name relative to exports/lib/ folder."""

    _plat_map = defaultdict(
        lambda: plat_name,
        {
            "win32": "win-x86",
        },
    )

    plat_split = _plat_map[plat_name].split("-", 1)
    if len(plat_split) != 2:
        return None
    plat_os, plat_arch = plat_split

    _os_map = defaultdict(
        lambda: plat_os,
        {
            "win": "windows",
        },
    )
    _lib_prefixes = defaultdict(
        lambda: "lib",
        {
            "windows": "",
        },
    )
    _lib_suffixes = defaultdict(
        lambda: ".so",
        {
            "windows": ".dll",
            "darwin": ".dylib",
        },
    )
    _arch_map = defaultdict(
        lambda: plat_arch,
        {
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
        },
    )

    processed_os = _os_map[plat_os]
    return (
        f"{processed_os}/{_arch_map[plat_arch]}/"
        f"{_lib_prefixes[processed_os]}{_libname}{_lib_suffixes[processed_os]}"
    )


# Adapted from https://stackoverflow.com/a/51794740
# You would say there would be a better way to do all of this, but I couldn't find it


class bdist_wheel(_bdist_wheel):
    user_options = _bdist_wheel.user_options + [
        ("exports-lib-path=", None, "path to exports/lib directory"),
    ]

    def initialize_options(self):
        super().initialize_options()
        self.exports_lib_path = "../../exports/lib"  # default

    def run(self):
        self.plat_name_supplied = True  # Force use platform

        libpath = getlibpath(self.plat_name)
        if not libpath:
            print(f"Unknown platform: {self.plat_name}")
            sys.exit(1)

        print(f"Building wheel for platform {self.plat_name}")

        # setuptools will only use paths inside the package for package_data, so we copy the library
        tmp_lib = shutil.copy(f"{self.exports_lib_path}/{libpath}", "src/lib/")
        _bdist_wheel.run(self)
        os.remove(tmp_lib)


setup(
    name="eduvpncommon",
    version="0.1.0",
    packages=["eduvpncommon"],
    python_requires=">=3.6",
    package_dir={"eduvpncommon": "src"},
    package_data={"eduvpncommon": [f"lib/*{_libname}*"]},
    cmdclass={"bdist_wheel": bdist_wheel},
)
