import pathlib
import platform
from collections import defaultdict
from ctypes import *
from enum import Enum

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
_lib = cdll.LoadLibrary(str(pathlib.Path(__file__).parent / "lib" / _libfile))


class _GoSlice(Structure):
    _fields_ = [("data", POINTER(c_char)), ("len", c_int64), ("cap", c_int64)]

    @staticmethod
    def make(bs: bytes) -> "_GoSlice":
        return _GoSlice((c_char * len(bs))(*bs), len(bs), len(bs))


_lib.Verify.argtypes, _lib.Verify.restype = [_GoSlice, _GoSlice, _GoSlice, c_uint64], c_int64
_lib.InsecureTestingSetExtraKey.argtypes, _lib.InsecureTestingSetExtraKey.restype = [_GoSlice], None


class VerifyErrorCode(Enum):
    ErrUnknownExpectedFileName = 1  # Unknown expected file name specified. The signature has not been verified.
    ErrInvalidSignature = 2  # Signature is invalid (for the expected file type).
    ErrInvalidSignatureUnknownKey = 3  # Signature was created with an unknown key and has not been verified.
    ErrTooOld = 4  # Signature timestamp smaller than specified minimum signing time (rollback).
    Unknown = -1  # Other unknown error.


class VerifyError(Exception):
    code: VerifyErrorCode
    code_int: int  # Original error code also for VerifyErrorCode.Unknown

    def __init__(self, err: int):
        assert err
        try:
            self.code = VerifyErrorCode(err)
        except ValueError:
            self.code = VerifyErrorCode.Unknown
        self.code_int = err

    def __str__(self):
        return \
            {
                VerifyErrorCode.ErrUnknownExpectedFileName: "unknown expected file name",
                VerifyErrorCode.ErrInvalidSignature: "invalid signature",
                VerifyErrorCode.ErrInvalidSignatureUnknownKey: "invalid signature (unknown key)",
                VerifyErrorCode.ErrTooOld: "replay of previous signature (rollback)",
            }[self.code] if self.code != VerifyErrorCode.Unknown else f"unknown verify error ({self.code_int})"


def verify(signature: bytes, signed_json: bytes, expected_file_name: str, min_sign_time: int) -> None:
    """
    Verifies the signature on the JSON server_list.json/organization_list.json file.
    If the function returns, the signature is valid for the given file type.

    :param signature: .minisig signature file contents.
    :param signed_json: Signed .json file contents.
    :param expected_file_name: The file type to be verified, one of "server_list.json" or "organization_list.json".
    :param min_sign_time: Minimum time for signature (UNIX timestamp, seconds). Should be set to at least the time of the previous signature.

    :raises VerifyException: If signature verification fails or expectedFileName is not one of the allowed values.
    """

    err = _lib.Verify(_GoSlice.make(signature), _GoSlice.make(signed_json),
                      _GoSlice.make(expected_file_name.encode()), min_sign_time)
    if err:
        raise VerifyError(err)


def _insecure_testing_set_extra_key(key_string: str) -> None:
    """Use for testing only, see Go documentation."""

    _lib.InsecureTestingSetExtraKey(_GoSlice.make(key_string.encode()))
