from ctypes import (CDLL, CFUNCTYPE, POINTER, Structure, c_char_p, c_int,
                    c_size_t, c_ulonglong, c_void_p, cast, pointer)
from typing import Any, Callable, Iterator, List, Optional, Tuple

from eduvpn_common.error import ErrorLevel, WrappedError


class cError(Structure):
    _fields_ = [
        ("level", c_int),
        ("traceback", c_char_p),
        ("cause", c_char_p),
    ]


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


class ConfigError(Structure):
    _fields_ = [("config", c_void_p), ("config_type", c_void_p), ("error", c_void_p)]


VPNStateChange = CFUNCTYPE(None, c_char_p, c_int, c_int, c_void_p)


def encode_args(args: List[Any], types: List[Any]) -> Iterator[Any]:
    for arg, t in zip(args, types):
        # c_char_p needs the str to be encoded to bytes
        if t is c_char_p:
            arg = arg.encode("utf-8")
        yield arg


def decode_res(res: Any):
    decode_map = {
        c_int: get_bool,
        c_void_p: get_error,
        DataError: get_data_error,
        ConfigError: get_config_error,
    }
    return decode_map.get(res, lambda lib, x: x)


def get_ptr_string(lib: CDLL, ptr: c_void_p) -> str:
    if ptr:
        string = cast(ptr, c_char_p).value
        lib.FreeString(ptr)
        if string:
            return string.decode("utf-8")
    return ""


def get_ptr_list_strings(
    lib: CDLL, strings: pointer, total_strings: int
) -> List[str]:
    if strings:
        strings_list = []
        for i in range(total_strings):
            strings_list.append(strings[i].decode("utf-8"))
        return strings_list
    return []


def get_error(lib: CDLL, ptr: c_void_p) -> Optional[WrappedError]:
    if not ptr:
        return None
    err = cast(ptr, POINTER(cError)).contents
    wrapped = WrappedError(
        err.traceback.decode("utf-8"), err.cause.decode("utf-8"), ErrorLevel(err.level)
    )
    lib.FreeError(ptr)
    return wrapped


def get_config_error(
    lib: CDLL, config_error: ConfigError
) -> Tuple[str, str, Optional[WrappedError]]:
    config = get_ptr_string(lib, config_error.config)
    config_type = get_ptr_string(lib, config_error.config_type)
    err = get_error(lib, config_error.error)
    return config, config_type, err


def get_data_error(
    lib: CDLL, data_error: DataError, data_conv: Callable = get_ptr_string
) -> Tuple[str, Optional[WrappedError]]:
    data = data_conv(lib, data_error.data)
    error = get_error(lib, data_error.error)
    return data, error


def get_bool(lib: CDLL, boolInt: c_int) -> bool:
    return boolInt == 1
