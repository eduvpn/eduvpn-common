from . import lib, GoSlice
from ctypes import *
from enum import Enum

lib.Verify.argtypes, lib.Verify.restype = [GoSlice, GoSlice, GoSlice, c_uint64], c_int64
lib.InsecureTestingSetExtraKey.argtypes, lib.InsecureTestingSetExtraKey.restype = [GoSlice], None


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

    err = lib.Verify(GoSlice.make(signature), GoSlice.make(signed_json),
                      GoSlice.make(expected_file_name.encode()), min_sign_time)
    if err:
        raise VerifyError(err)


def _insecure_testing_set_extra_key(key_string: str) -> None:
    """Use for testing only, see Go documentation."""

    lib.InsecureTestingSetExtraKey(GoSlice.make(key_string.encode()))
