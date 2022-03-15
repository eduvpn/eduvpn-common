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

# Data types
class DataError(Structure):
    _fields_ = [('data', c_void_p),
                ('error', c_int64)]

GOCB_StateChange = CFUNCTYPE(None, c_char_p, c_char_p)


# Exposed functions
lib.Register.argtypes, lib.Register.restype = [c_char_p, c_char_p, GOCB_StateChange], None
# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.InitializeOAuth.argtypes, lib.InitializeOAuth.restype = [], c_void_p
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None
