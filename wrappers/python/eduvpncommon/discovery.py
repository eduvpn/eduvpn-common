from . import lib, GoSlice, DataError
from ctypes import *
from typing import Callable
from enum import Enum

# We have to use c_void_p instead of c_char_p to free it properly
# See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
lib.GetOrganizationsList.argtypes, lib.GetOrganizationsList.restype = [], DataError
lib.GetServersList.argtypes, lib.GetServersList.restype = [], DataError
lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None

lib.Verify.argtypes, lib.Verify.restype = [GoSlice, GoSlice, GoSlice, c_uint64], c_int64
lib.InsecureTestingSetExtraKey.argtypes, lib.InsecureTestingSetExtraKey.restype = [GoSlice], None

def getList(func: Callable) -> str:
    dataError = func()
    ptr = dataError.data
    error = dataError.error
    body = ""
    if not error:
        body = str(cast(ptr, c_char_p).value)
    lib.FreeString(ptr)
    if error:
        raise RequestError(error)
    return body

def GetOrganizationsList() -> str:
    return getList(lib.GetOrganizationsList)

def GetServersList() -> str:
    return getList(lib.GetServersList)


class GoError(Exception):
    message_dict: dict
    code: Enum | None

    def __init__(self, err: Enum, messages: dict):
        assert err
        try:
            self.code = err
        except ValueError:
            self.code = None
        self.message_dict = messages

    def __str__(self):
        return self.message_dict[self.code] if self.code in self.message_dict else f"unknown error ({self.code})"


class RequestErrorCode(Enum):
    ErrRequestFileError = 1  # The request for the file has failed.
    ErrVerifySigError = 2  # The signature failed to verify.
    Unknown = -1  # Other unknown error.

class RequestError(GoError):
    def __init__(self, err: int):
        super().__init__(RequestErrorCode(err),
            {
                RequestErrorCode.ErrRequestFileError: "file request error",
                RequestErrorCode.ErrVerifySigError: "signature verify error",
            })


class VerifyErrorCode(Enum):
    ErrUnknownExpectedFileName = 1  # Unknown expected file name specified. The signature has not been verified.
    ErrInvalidSignature = 2  # Signature is invalid (for the expected file type).
    ErrInvalidSignatureUnknownKey = 3  # Signature was created with an unknown key and has not been verified.
    ErrTooOld = 4  # Signature timestamp smaller than specified minimum signing time (rollback).
    Unknown = -1  # Other unknown error.

class VerifyError(GoError):
    def __init__(self, err: int):
        super().__init__(VerifyErrorCode(err),
            {
                VerifyErrorCode.ErrUnknownExpectedFileName: "unknown expected file name",
                VerifyErrorCode.ErrInvalidSignature: "invalid signature",
                VerifyErrorCode.ErrInvalidSignatureUnknownKey: "invalid signature (unknown key)",
                VerifyErrorCode.ErrTooOld: "replay of previous signature (rollback)",
            })


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

    err = lib.Verify(GoSlice.make(signature), GoSlice.make(signed_json),
                      GoSlice.make(expected_file_name.encode()), min_sign_time)
    if err:
        raise VerifyError(err)


def _insecure_testing_set_extra_key(key_string: str) -> None:
    """Use for testing only, see Go documentation."""

    lib.InsecureTestingSetExtraKey(GoSlice.make(key_string.encode()))
