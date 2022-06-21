from ctypes import *
from collections import defaultdict
import pathlib
import platform
from typing import Tuple

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


class MultipleDataError(Structure):
    _fields_ = [("data", c_void_p), ("other_data", c_void_p), ("error", c_void_p)]


VPNStateChange = CFUNCTYPE(None, c_char_p, c_char_p, c_char_p)

# Exposed functions
# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.GetConnectConfig.argtypes, lib.GetConnectConfig.restype = [
    c_char_p,
    c_char_p,
    c_int,
    c_int,
], MultipleDataError
lib.Deregister.argtypes, lib.Deregister.restype = [c_char_p], c_void_p
lib.Register.argtypes, lib.Register.restype = [
    c_char_p,
    c_char_p,
    VPNStateChange,
    c_int,
], c_void_p
lib.GetOrganizationsList.argtypes, lib.GetOrganizationsList.restype = [
    c_char_p
], DataError
lib.GetServersList.argtypes, lib.GetServersList.restype = [c_char_p], DataError
lib.CancelOAuth.argtypes, lib.CancelOAuth.restype = [c_char_p], c_void_p
lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p, c_char_p], c_void_p
lib.SetConnected.argtypes, lib.SetConnected.restype = [c_char_p], c_void_p
lib.SetDisconnected.argtypes, lib.SetDisconnected.restype = [c_char_p], c_void_p
lib.GetIdentifier.argtypes, lib.GetIdentifier.restype = [c_char_p], DataError
lib.SetIdentifier.argtypes, lib.SetIdentifier.restype = [c_char_p, c_char_p], c_void_p
lib.SetSearchServer.argtypes, lib.SetSearchServer.restype = [c_char_p], c_void_p
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None


def GetPtrString(ptr: c_void_p) -> str:
    if ptr:
        string = cast(ptr, c_char_p).value
        lib.FreeString(ptr)
        if string:
            return string.decode()
    return ""


def GetDataError(data_error: DataError) -> Tuple[str, str]:
    data = GetPtrString(data_error.data)
    error = GetPtrString(data_error.error)
    return data, error


def GetMultipleDataError(
    multiple_data_error: MultipleDataError,
) -> Tuple[str, str, str]:
    data = GetPtrString(multiple_data_error.data)
    other_data = GetPtrString(multiple_data_error.other_data)
    error = GetPtrString(multiple_data_error.error)
    return data, other_data, error
