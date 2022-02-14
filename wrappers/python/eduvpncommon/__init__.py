from ctypes import *
from collections import defaultdict
import pathlib
import platform

_lib_prefixes = defaultdict(lambda: "lib", {
    "windows": "",
})

_lib_suffixes = defaultdict(lambda: ".so", {
    "windows": ".dll",
    "darwin": ".dylib",
})

_os = platform.system().lower()

_libname = "eduvpn_common"
_libfile = f"{_lib_prefixes[_os]}{_libname}{_lib_suffixes[_os]}"
# Library should have been copied to the lib/ folder
lib = cdll.LoadLibrary(str(pathlib.Path(__file__).parent / "lib" / _libfile))


class GoSlice(Structure):
    _fields_ = [("data", POINTER(c_char)), ("len", c_int64), ("cap", c_int64)]

    @staticmethod
    def make(bs: bytes) -> "GoSlice":
        return GoSlice((c_char * len(bs))(*bs), len(bs), len(bs)) # type: ignore


class DataError(Structure):
    _fields_ = [('data', c_void_p),
                ('error', c_int64)]
