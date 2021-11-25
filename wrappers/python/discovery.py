import platform
from ctypes import *
from enum import Enum

_lib_suffixes = {
    "windows": ".dll",
    "linux": ".so",
    "darwin": ".dylib",
}

_arch = platform.machine().lower()
_arch = \
    {
        "aarch64_be": "arm64",
        "aarch64": "arm64",
        "armv8b": "arm64",
        "armv8l": "arm64",
        "x86": "386",
        "x86pc": "386",
        "i86pc": "386",
        "i386": "386",
        "i686": "386",
        "x86_64": "amd64",
        "i686-64": "amd64",
    }.get(_arch, _arch)

_os = platform.system().lower()

_lib = cdll.LoadLibrary(f"../../exports/{_os}/{_arch}/eduvpn_verify{_lib_suffixes[_os]}")


class GoSlice(Structure):
    _fields_ = [("data", POINTER(c_char)), ("len", c_int64), ("cap", c_int64)]

    @staticmethod
    def make(bs: bytes) -> "GoSlice":
        return GoSlice((c_char * len(bs))(*bs), len(bs), len(bs))


_lib.Verify.argtypes, _lib.Verify.restype = [GoSlice, GoSlice, GoSlice, c_uint64], c_int64
_lib.InsecureTestingSetExtraKey.argtypes, _lib.InsecureTestingSetExtraKey.restype = [GoSlice], None


class VerifyErrorCode(Enum):
    ErrUnknownExpectedFileName = 1  # Expected file name is not one of the recognized values.
    ErrInvalidSignature = 2  # Signature is invalid (for the expected file type).
    ErrInvalidSignatureUnknownKey = 3  # Signature was created with an unknown key and has not been verified.
    ErrTooOld = 4  # Signature has a timestamp lower than the specified minimum signing time.
    Unknown = -1  # Other unknown error


class VerifyError(Exception):
    code: VerifyErrorCode
    code_int: int  # Original error code also for VerifyErrorCode.Unknown

    def __init__(self, err: int):
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
    If the function returns the signature is valid for the given file type.
    :param signature: .minisig signature file contents.
    :param signed_json: Signed .json file contents.
    :param expected_file_name: The file type to be verified, one of "server_list.json" or "organization_list.json".
    :param min_sign_time: Minimum time for signature. Should be set to at least the time in a previously retrieved file.

    :raises VerifyException: If signature verification fails or expectedFileName is not one of the allowed values.
    """

    err = _lib.Verify(GoSlice.make(signature), GoSlice.make(signed_json),
                      GoSlice.make(expected_file_name.encode()), min_sign_time)
    if err:
        raise VerifyError(err)


def _insecure_testing_set_extra_key(key_string: str) -> None:
    """Use for testing only, see Go documentation."""

    _lib.InsecureTestingSetExtraKey(GoSlice.make(key_string.encode()))
