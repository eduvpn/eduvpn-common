from ctypes import *
from collections import defaultdict
import pathlib
import platform

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

_os = platform.system().lower()

_libname = "eduvpn_common"
_libfile = f"{_lib_prefixes[_os]}{_libname}{_lib_suffixes[_os]}"
# Library should have been copied to the lib/ folder
lib = cdll.LoadLibrary(str(pathlib.Path(__file__).parent / "lib" / _libfile))


class DataError(Structure):
    _fields_ = [("data", c_void_p), ("error", c_void_p)]


VPNStateChange = CFUNCTYPE(None, c_char_p, c_char_p, c_char_p)

# Exposed functions
lib.Connect.argtypes, lib.Connect.restype = [c_char_p, c_char_p], DataError
lib.Deregister.argtypes, lib.Deregister.restype = [c_char_p], c_void_p
lib.Register.argtypes, lib.Register.restype = [c_char_p, c_char_p, VPNStateChange, c_int], c_void_p
lib.GetOrganizationsList.argtypes, lib.GetOrganizationsList.restype = [c_char_p], DataError
lib.GetServersList.argtypes, lib.GetServersList.restype = [c_char_p], DataError
lib.CancelOAuth.argtypes, lib.CancelOAuth.restype = [c_char_p], c_void_p
lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p, c_char_p], c_void_p
# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.SetConnected.argtypes, lib.SetConnected.restype = [c_char_p], c_void_p
lib.SetDisconnected.argtypes, lib.SetDisconnected.restype = [c_char_p], c_void_p
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None


def GetPtrString(ptr):
    if ptr:
        string = cast(ptr, c_char_p).value
        lib.FreeString(ptr)
        if string:
            return string.decode()
    return ""


def GetDataError(data_error):
    data = GetPtrString(data_error.data)
    error = GetPtrString(data_error.error)
    return data, error
