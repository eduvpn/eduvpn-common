#!/usr/bin/env python3

import os
import pathlib
import shutil
import sys

from setuptools import setup
from wheel.bdist_wheel import bdist_wheel as _bdist_wheel

# You would say there would be a better way to do all of this, but I couldn't find it

class bdist_wheel(_bdist_wheel):
    def run(self):
        self.plat_name_supplied = True  # Force use platform

        libpath = {
            # TODO arm may be incorrect; also add more
            "win-amd64": "windows/amd64/eduvpn_verify.dll",
            "win32": "windows/386/eduvpn_verify.dll",
            "win-arm32": "windows/arm/eduvpn_verify.dll",
            "win-arm64": "windows/arm64/eduvpn_verify.dll",
            "linux-x86_64": "linux/amd64/libeduvpn_verify.so",
            "linux-i386": "linux/386/libeduvpn_verify.so",
            "linux-i686": "linux/386/libeduvpn_verify.so",
            "linux-arm": "linux/arm/libeduvpn_verify.so",
            "linux-aarch64": "linux/arm64/libeduvpn_verify.so",
        }

        if self.plat_name not in libpath:
            print(f"Unknown platform: {self.plat_name}")
            sys.exit(1)

        print(f"Building wheel for platform {self.plat_name}")

        shutil.copy2(f"../../exports/{libpath[self.plat_name]}", "eduvpncommon/lib/")
        _bdist_wheel.run(self)
        os.remove(f"eduvpncommon/lib/{pathlib.Path(libpath[self.plat_name]).name}")


setup(
    name="eduvpncommon",
    version="0.1.0",
    packages=["eduvpncommon"],
    python_requires=">=3.6",
    package_data={"eduvpncommon": ["lib/*eduvpn_verify*"]},
    cmdclass={"bdist_wheel": bdist_wheel},
)
