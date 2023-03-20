from ctypes import (CDLL, CFUNCTYPE, Structure, c_char_p, c_int, c_ulonglong,
                    c_void_p, cast)
from typing import Any, Iterator, List, Tuple


class DataError(Structure):
    """The C type that represents a tuple of data and error as returned by the Go library

    :meta private:
    """

    _fields_ = [("data", c_void_p), ("error", c_void_p)]


# The type for a Go state change callback
VPNStateChange = CFUNCTYPE(c_int, c_int, c_int, c_char_p)
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
        c_void_p: get_ptr_string,
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


def get_data_error(
    lib: CDLL, data_error: DataError, data_conv: Any = get_ptr_string
) -> Tuple[str, str]:
    """Convert a C data+error structure to a Python usable data+error structure

    :param lib: CDLL: The Go shared library
    :param data_error: DataError: The data error C structure
    :param data_conv: Any: The function to convert the data

    :meta private:

    :return: The data and error
    :rtype: Tuple[str, str]
    """
    data = data_conv(lib, data_error.data)
    error = get_ptr_string(lib, data_error.error)
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
