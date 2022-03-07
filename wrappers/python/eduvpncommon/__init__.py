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


# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.Register.argtypes, lib.Register.restype = [c_char_p, c_char_p], None
lib.InitializeOAuth.argtypes, lib.InitializeOAuth.restype = [], c_void_p
lib.GetOrganizationsList.argtypes, lib.GetOrganizationsList.restype = [], DataError
lib.GetServersList.argtypes, lib.GetServersList.restype = [], DataError
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None
lib.Verify.argtypes, lib.Verify.restype = [GoSlice, GoSlice, GoSlice, c_uint64], c_int64
lib.InsecureTestingSetExtraKey.argtypes, lib.InsecureTestingSetExtraKey.restype = [GoSlice], None

