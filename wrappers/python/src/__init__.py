from ctypes import *
from collections import defaultdict
from enum import Enum
import pathlib
import platform
from typing import Tuple, Optional
import json
from typing import List

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


class cServerLocations(Structure):
    _fields_ = [("locations", POINTER(c_char_p)), ("total_locations", c_size_t)]


class cDiscoveryOrganization(Structure):
    _fields_ = [
        ("display_name", c_char_p),
        ("org_id", c_char_p),
        ("secure_internet_home", c_char_p),
        ("keyword_list", c_char_p),
    ]


class cDiscoveryOrganizations(Structure):
    _fields_ = [
        ("version", c_ulonglong),
        ("organizations", POINTER(POINTER(cDiscoveryOrganization))),
        ("total_organizations", c_size_t),
    ]


class cDiscoveryServer(Structure):
    _fields_ = [
        ("authentication_url_template", c_char_p),
        ("base_url", c_char_p),
        ("country_code", c_char_p),
        ("display_name", c_char_p),
        ("keyword_list", c_char_p),
        ("public_key_list", POINTER(c_char_p)),
        ("total_public_keys", c_size_t),
        ("server_type", c_char_p),
        ("support_contact", POINTER(c_char_p)),
        ("total_support_contact", c_size_t),
    ]


class cDiscoveryServers(Structure):
    _fields_ = [
        ("version", c_ulonglong),
        ("servers", POINTER(POINTER(cDiscoveryServer))),
        ("total_servers", c_size_t),
    ]


class cServerProfile(Structure):
    _fields_ = [
        ("identifier", c_char_p),
        ("display_name", c_char_p),
        ("default_gateway", c_int),
    ]


class cServerProfiles(Structure):
    _fields_ = [
        ("current", c_int),
        ("profiles", POINTER(POINTER(cServerProfile))),
        ("total_profiles", c_size_t),
    ]


class cServer(Structure):
    _fields_ = [
        ("identifier", c_char_p),
        ("display_name", c_char_p),
        ("server_type", c_char_p),
        ("country_code", c_char_p),
        ("support_contact", POINTER(c_char_p)),
        ("total_support_contact", c_size_t),
        ("profiles", POINTER(cServerProfiles)),
        ("expire_time", c_ulonglong),
    ]


class cServers(Structure):
    _fields_ = [
        ("custom_servers", POINTER(POINTER(cServer))),
        ("total_custom", c_size_t),
        ("institute_servers", POINTER(POINTER(cServer))),
        ("total_institute", c_size_t),
        ("secure_internet", POINTER(cServer)),
    ]


class DataError(Structure):
    _fields_ = [("data", c_void_p), ("error", c_void_p)]


VPNStateChange = CFUNCTYPE(None, c_char_p, c_int, c_int, c_void_p)

# Exposed functions
# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.RemoveSecureInternet.argtypes, lib.RemoveSecureInternet.restype = [
    c_char_p
], c_void_p
lib.RemoveInstituteAccess.argtypes, lib.RemoveInstituteAccess.restype = [
    c_char_p,
    c_char_p,
], c_void_p
lib.RemoveCustomServer.argtypes, lib.RemoveCustomServer.restype = [
    c_char_p,
    c_char_p,
], c_void_p
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
lib.Deregister.argtypes, lib.Deregister.restype = [c_char_p], None
lib.Register.argtypes, lib.Register.restype = [
    c_char_p,
    c_char_p,
    VPNStateChange,
    c_int,
], c_void_p
lib.GetDiscoOrganizations.argtypes, lib.GetDiscoOrganizations.restype = [
    c_char_p
], c_void_p
lib.GetDiscoServers.argtypes, lib.GetDiscoServers.restype = [c_char_p], c_void_p
lib.GoBack.argtypes, lib.GoBack.restype = [c_char_p], None
lib.CancelOAuth.argtypes, lib.CancelOAuth.restype = [c_char_p], c_void_p
lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p, c_char_p], c_void_p
lib.ChangeSecureLocation.argtypes, lib.ChangeSecureLocation.restype = [
    c_char_p
], c_void_p
lib.SetSecureLocation.argtypes, lib.SetSecureLocation.restype = [
    c_char_p,
    c_char_p,
], c_void_p
lib.SetConnected.argtypes, lib.SetConnected.restype = [c_char_p], c_void_p
lib.SetDisconnecting.argtypes, lib.SetDisconnecting.restype = [c_char_p], c_void_p
lib.SetConnecting.argtypes, lib.SetConnecting.restype = [c_char_p], c_void_p
lib.SetDisconnected.argtypes, lib.SetDisconnected.restype = [c_char_p, c_int], c_void_p
lib.SetSearchServer.argtypes, lib.SetSearchServer.restype = [c_char_p], c_void_p
lib.ShouldRenewButton.argtypes, lib.ShouldRenewButton.restype = [], int
lib.RenewSession.argtypes, lib.RenewSession.restype = [c_char_p], c_void_p
lib.FreeProfiles.argtypes, lib.FreeProfiles.restype = [c_void_p], None
lib.FreeSecureLocations.argtypes, lib.FreeSecureLocations.restype = [c_void_p], None
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None
lib.FreeDiscoOrganizations.argtypes, lib.FreeDiscoOrganizations.restype = [
    c_void_p
], None
lib.FreeDiscoServers.argtypes, lib.FreeDiscoServers.restype = [c_void_p], None
lib.FreeServer.argtypes, lib.FreeServer.restype = [c_void_p], None
lib.FreeServers.argtypes, lib.FreeServers.restype = [c_void_p], None
lib.InFSMState.argtypes, lib.InFSMState.restype = [c_void_p, c_int], int
lib.GetSavedServers.argtypes, lib.GetSavedServers.restype = [c_char_p], c_void_p


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


def get_ptr_list_strings(
    strings: POINTER(c_char_p), total_strings: c_size_t
) -> List[str]:
    if strings:
        strings_list = []
        for i in range(total_strings):
            strings_list.append(strings[i].decode("utf-8"))
        return strings_list
    return []


def get_ptr_error(ptr: c_void_p) -> Optional[WrappedError]:
    error_string = get_ptr_string(ptr)

    if not error_string:
        return None

    error_json = json.loads(error_string)

    if not error_json:
        return None

    if "level" not in error_json:
        return error_string
    level = error_json["level"]
    traceback = error_json["traceback"]
    cause = error_json["cause"]
    return WrappedError(traceback, cause, ErrorLevel(level))


def get_error(ptr: c_void_p) -> str:
    error = get_ptr_error(ptr)
    if not error:
        return ""

    if not isinstance(error, WrappedError):
        return error
    return error.cause


def get_data_error(data_error: DataError) -> Tuple[str, str]:
    data = get_ptr_string(data_error.data)
    error = get_error(data_error.error)
    return data, error


def get_bool(boolInt: c_int) -> bool:
    return boolInt == 1


decode_map = {
    c_int: get_bool,
    c_void_p: get_error,
    DataError: get_data_error,
}
