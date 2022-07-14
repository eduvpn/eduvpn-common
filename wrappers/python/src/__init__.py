from ctypes import *
from collections import defaultdict
from enum import Enum
import pathlib
import platform
from typing import Tuple, Optional
import json

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

lib = None

# Try to load in the normal path
try:
    lib = cdll.LoadLibrary(_libfile)
# Otherwise, library should have been copied to the lib/ folder
except:
    lib = cdll.LoadLibrary(str(pathlib.Path(__file__).parent / "lib" / _libfile))

class ErrorLevel(Enum):
    ERR_OTHER = 0
    ERR_INFO = 1

class DataError(Structure):
    _fields_ = [("data", c_void_p), ("error", c_void_p)]


VPNStateChange = CFUNCTYPE(None, c_char_p, c_char_p, c_char_p, c_char_p)

# Exposed functions
# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.GetConfigSecureInternet.argtypes, lib.GetConfigSecureInternet.restype = [
    c_char_p,
    c_char_p,
    c_int,
], DataError
lib.GetConfigInstituteAccess.argtypes, lib.GetConfigInstituteAccess.restype = [
    c_char_p,
    c_char_p,
    c_int,
], DataError
lib.GetConfigCustomServer.argtypes, lib.GetConfigCustomServer.restype = [
    c_char_p,
    c_char_p,
    c_int,
], DataError
lib.Deregister.argtypes, lib.Deregister.restype = [c_char_p], c_void_p
lib.Register.argtypes, lib.Register.restype = [
    c_char_p,
    c_char_p,
    VPNStateChange,
    c_int,
], c_void_p
lib.GetDiscoOrganizations.argtypes, lib.GetDiscoOrganizations.restype = [
    c_char_p
], DataError
lib.GetDiscoServers.argtypes, lib.GetDiscoServers.restype = [c_char_p], DataError
lib.GoBack.argtypes, lib.GoBack.restype = [c_char_p], None
lib.CancelOAuth.argtypes, lib.CancelOAuth.restype = [c_char_p], c_void_p
lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p, c_char_p], c_void_p
lib.SetSecureLocation.argtypes, lib.SetSecureLocation.restype = [
    c_char_p,
    c_char_p,
], c_void_p
lib.SetConnected.argtypes, lib.SetConnected.restype = [c_char_p], c_void_p
lib.SetDisconnected.argtypes, lib.SetDisconnected.restype = [c_char_p], c_void_p
lib.GetIdentifier.argtypes, lib.GetIdentifier.restype = [c_char_p], DataError
lib.SetIdentifier.argtypes, lib.SetIdentifier.restype = [c_char_p, c_char_p], c_void_p
lib.SetSearchServer.argtypes, lib.SetSearchServer.restype = [c_char_p], c_void_p
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None


class WrappedError:
    def __init__(self, traceback: str, cause: str, level: ErrorLevel):
        self.traceback = traceback
        self.cause = cause
        self.level = level


def encode_args(args, types):
    for arg, t in zip(args, types):
        # c_char_p needs the str to be encoded to bytes
        if t is c_char_p:
            arg = arg.encode("utf-8")
        yield arg


def decode_res(t):
    return decode_map.get(t, lambda x: x)


def get_ptr_string(ptr: c_void_p) -> str:
    if ptr:
        string = cast(ptr, c_char_p).value
        lib.FreeString(ptr)
        if string:
            return string.decode()
    return ""

def get_ptr_error(ptr: c_void_p) -> Optional[WrappedError]:
    error_string = get_ptr_string(ptr)

    if not error_string:
        return None

    error_json = json.loads(error_string)

    if not error_json:
        return None

    level = error_json["level"]
    traceback = error_json["traceback"]
    cause = error_json["cause"]
    return WrappedError(traceback, cause, ErrorLevel(level))

def get_error(ptr: c_void_p) -> str:
    error = get_ptr_error(ptr)
    if not error:
        return ""
    return error.traceback

def get_data_error(data_error: DataError) -> Tuple[str, str]:
    data = get_ptr_string(data_error.data)
    error = get_error(data_error.error)
    return data, error


decode_map = {
    c_void_p: get_error,
    DataError: get_data_error,
}
