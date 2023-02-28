from ctypes import (
    CDLL,
    CFUNCTYPE,
    POINTER,
    Structure,
    c_char_p,
    c_int,
    c_size_t,
    c_ulonglong,
    c_void_p,
    cast,
    pointer,
)
from typing import Any, Callable, Iterator, List, Optional, Tuple

from eduvpn_common.error import WrappedError


class cToken(Structure):
    """The C type that represents the Token as forwarded to the Go library

    :meta private:
    """

    _fields_ = [
        ("access", c_char_p),
        ("refresh", c_char_p),
        ("expired", c_ulonglong),
    ]


class cConfig(Structure):
    """The C type that represents the data that gets by the Go library returned when a config is obtained

    :meta private:
    """

    _fields_ = [
        ("config", c_char_p),
        ("config_type", c_char_p),
        ("token", POINTER(cToken)),
    ]


class cError(Structure):
    """The C type that represents the Error as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("traceback", c_char_p),
        ("cause", c_char_p),
    ]


class cServerLocations(Structure):
    """The C type that represents the Server Locations as returned by the Go library

    :meta private:
    """

    _fields_ = [("locations", POINTER(c_char_p)), ("total_locations", c_size_t)]


class cDiscoveryOrganization(Structure):
    """The C type that represents a Discovery Organization as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("display_name", c_char_p),
        ("org_id", c_char_p),
        ("secure_internet_home", c_char_p),
        ("keyword_list", c_char_p),
    ]


class cDiscoveryOrganizations(Structure):
    """The C type that represents Discovery Organizations as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("version", c_ulonglong),
        ("organizations", POINTER(POINTER(cDiscoveryOrganization))),
        ("total_organizations", c_size_t),
    ]


class cDiscoveryServer(Structure):
    """The C type that represents a Discovery Server as returned by the Go library

    :meta private:
    """

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
    """The C type that represents Discovery Servers as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("version", c_ulonglong),
        ("servers", POINTER(POINTER(cDiscoveryServer))),
        ("total_servers", c_size_t),
    ]


class cServerProfile(Structure):
    """The C type that represents a Server Profile as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("identifier", c_char_p),
        ("display_name", c_char_p),
        ("default_gateway", c_int),
    ]


class cServerProfiles(Structure):
    """The C type that represents Server Profiles as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("current", c_int),
        ("profiles", POINTER(POINTER(cServerProfile))),
        ("total_profiles", c_size_t),
    ]


class cServer(Structure):
    """The C type that represents a Server as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("identifier", c_char_p),
        ("display_name", c_char_p),
        ("server_type", c_char_p),
        ("country_code", c_char_p),
        ("support_contact", POINTER(c_char_p)),
        ("total_support_contact", c_size_t),
        ("locations", POINTER(cServerLocations)),
        ("profiles", POINTER(cServerProfiles)),
        ("expire_time", c_ulonglong),
    ]


class cServers(Structure):
    """The C type that represents Servers as returned by the Go library

    :meta private:
    """

    _fields_ = [
        ("custom_servers", POINTER(POINTER(cServer))),
        ("total_custom", c_size_t),
        ("institute_servers", POINTER(POINTER(cServer))),
        ("total_institute", c_size_t),
        ("secure_internet", POINTER(cServer)),
    ]


class DataError(Structure):
    """The C type that represents a tuple of data and error as returned by the Go library

    :meta private:
    """

    _fields_ = [("data", c_void_p), ("error", c_void_p)]


# The type for a Go state change callback
VPNStateChange = CFUNCTYPE(c_int, c_char_p, c_int, c_int, c_void_p)
ReadRxBytes = CFUNCTYPE(c_ulonglong)


def encode_args(args: List[Any], types: List[Any]) -> Iterator[Any]:
    """Encode the arguments ready to be used by the Go library

    :param args: List[Any]: The list of arguments
    :param types: List[Any]: The list of the types of the arguments

    :meta private:

    :return: The arg generator
    :rtype: Iterator[Any]
    """
    for arg, t in zip(args, types):
        # c_char_p needs the str to be encoded to bytes
        encode_map = {
            c_char_p: lambda x: x.encode("utf-8"),
        }
        if t in encode_map:
            arg = encode_map[t](arg)
        yield arg


def decode_res(res: Any) -> Any:
    """Decode a result as obtained by the Go library

    :param res: Any: The result

    :meta private:

    :return: The argument decoded
    :rtype: Any
    """
    decode_map = {
        c_int: get_bool,
        c_void_p: get_error,
        DataError: get_data_error,
    }
    return decode_map.get(res, lambda lib, x: x)


def get_ptr_string(lib: CDLL, ptr: c_void_p) -> str:
    """Convert a C string pointer to a Python usable string.
    This makes sure to free all memory allocated by the Go library

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The pointer to the C string

    :meta private:

    :return: The string converted to Python
    :rtype: str
    """
    if ptr:
        string = cast(ptr, c_char_p).value
        lib.FreeString(ptr)
        if string:
            return string.decode("utf-8")
    return ""


def get_ptr_list_strings(lib: CDLL, strings: pointer, total_strings: int) -> List[str]:
    """Convert a list of C strings to a Python usable list of strings
    This list is not freed here but is later freed in the Go library for convenience

    :param lib: CDLL: The Go shared library
    :param strings: pointer: The C pointer to the strings list
    :param total_strings: int: The total strings in the list

    :meta private:

    :return: The list of strings converted to Python
    :rtype: List[str]
    """
    if strings:
        strings_list = []
        for i in range(total_strings):
            strings_list.append(strings[i].decode("utf-8"))
        return strings_list
    return []


def get_error(lib: CDLL, ptr: c_void_p) -> Optional[WrappedError]:
    """Convert a C error structure to a Python usable error structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The pointer to the C error struct

    :meta private:

    :return: The error if there is one
    :rtype: Optional[WrappedError]
    """
    if not ptr:
        return None
    err = cast(ptr, POINTER(cError)).contents
    wrapped = WrappedError(err.traceback.decode("utf-8"), err.cause.decode("utf-8"))
    lib.FreeError(ptr)
    return wrapped


def get_data_error(
    lib: CDLL, data_error: DataError, data_conv: Callable = get_ptr_string
) -> Tuple[Any, Optional[WrappedError]]:
    """Convert a C data+error structure to a Python usable data+error structure

    :param lib: CDLL: The Go shared library
    :param data_error: DataError: The data error C structure
    :param data_conv: Callable:  (Default value = get_ptr_string): The function that converts the data part

    :meta private:

    :return: The data and optional error
    :rtype: Tuple[Any, Optional[WrappedError]]
    """
    data = data_conv(lib, data_error.data)
    error = get_error(lib, data_error.error)
    return data, error


def get_bool(lib: CDLL, boolInt: c_int) -> bool:
    """Get a bool from the Go shared library. Essentially just checking if an int represents 'True'

    :param lib: CDLL: The Go shared library
    :param boolInt: c_int: The C integer that needs to be converted to the Python bool

    :meta private:

    :return: The boolean converted to Python
    :rtype: bool
    """
    return boolInt == 1
