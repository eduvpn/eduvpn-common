import pathlib
import platform
from collections import defaultdict
from ctypes import CDLL, c_char_p, c_int, c_void_p, cdll

from eduvpn_common import __version__
from eduvpn_common.types import (
    cToken,
    DataError,
    ReadRxBytes,
    VPNStateChange,
)


def load_lib() -> CDLL:
    """The function that loads the Go shared library

    :meta private:

    :return: The Go shared library loaded with cdll.LoadLibrary from ctypes
    :rtype: CDLL
    """
    lib_prefixes = defaultdict(
        lambda: "lib",
        {
            "windows": "",
        },
    )

    lib_suffixes = defaultdict(
        lambda: ".so",
        {
            "windows": ".dll",
            "darwin": ".dylib",
        },
    )

    os = platform.system().lower()

    libname = "eduvpn_common"
    libfile = f"{lib_prefixes[os]}{libname}-{__version__}{lib_suffixes[os]}"

    lib = None

    # Try to load in the normal path
    try:
        lib = cdll.LoadLibrary(libfile)
        # Otherwise, library should have been copied to the lib/ folder
    except:
        lib = cdll.LoadLibrary(str(pathlib.Path(__file__).parent / "lib" / libfile))

    return lib


def initialize_functions(lib: CDLL) -> None:
    """Initializes the Go shared library functions

    :param lib: CDLL: The Go shared library

    :meta private:
    """
    # Exposed functions
    # We have to use c_void_p instead of c_char_p to free it properly
    # See https://stackoverflow.com/questions/13445568/python-ctypes-how-to-free-memory-getting-invalid-pointer-error
    lib.CancelOAuth.argtypes, lib.CancelOAuth.restype = [c_char_p], c_void_p
    lib.ChangeSecureLocation.argtypes, lib.ChangeSecureLocation.restype = [
        c_char_p
    ], c_void_p
    lib.Deregister.argtypes, lib.Deregister.restype = [c_char_p], None
    lib.FreeConfig.argtypes, lib.FreeConfig.restype = [c_void_p], None
    lib.FreeDiscoOrganizations.argtypes, lib.FreeDiscoOrganizations.restype = [
        c_void_p
    ], None
    lib.FreeDiscoServers.argtypes, lib.FreeDiscoServers.restype = [c_void_p], None
    lib.FreeError.argtypes, lib.FreeError.restype = [c_void_p], None
    lib.FreeProfiles.argtypes, lib.FreeProfiles.restype = [c_void_p], None
    lib.FreeSecureLocations.argtypes, lib.FreeSecureLocations.restype = [c_void_p], None
    lib.FreeServer.argtypes, lib.FreeServer.restype = [c_void_p], None
    lib.FreeServers.argtypes, lib.FreeServers.restype = [c_void_p], None
    lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None
    lib.GetConfigCustomServer.argtypes, lib.GetConfigCustomServer.restype = [
        c_char_p,
        c_char_p,
        c_int,
        cToken,
    ], DataError
    lib.GetConfigInstituteAccess.argtypes, lib.GetConfigInstituteAccess.restype = [
        c_char_p,
        c_char_p,
        c_int,
        cToken,
    ], DataError
    lib.GetConfigSecureInternet.argtypes, lib.GetConfigSecureInternet.restype = [
        c_char_p,
        c_char_p,
        c_int,
        cToken,
    ], DataError
    lib.GetDiscoOrganizations.argtypes, lib.GetDiscoOrganizations.restype = [
        c_char_p
    ], DataError
    lib.GetDiscoServers.argtypes, lib.GetDiscoServers.restype = [c_char_p], DataError
    lib.GetCurrentServer.argtypes, lib.GetCurrentServer.restype = [c_char_p], DataError
    lib.GetSavedServers.argtypes, lib.GetSavedServers.restype = [c_char_p], DataError
    lib.GoBack.argtypes, lib.GoBack.restype = [c_char_p], None
    lib.InFSMState.argtypes, lib.InFSMState.restype = [c_void_p, c_int], int
    lib.Register.argtypes, lib.Register.restype = [
        c_char_p,
        c_char_p,
        c_char_p,
        c_char_p,
        VPNStateChange,
        c_int,
    ], c_void_p
    lib.RemoveCustomServer.argtypes, lib.RemoveCustomServer.restype = [
        c_char_p,
        c_char_p,
    ], c_void_p
    lib.AddInstituteAccess.argtypes, lib.AddInstituteAccess.restype = [
        c_char_p,
        c_char_p,
    ], c_void_p
    (
        lib.AddSecureInternetHomeServer.argtypes,
        lib.AddSecureInternetHomeServer.restype,
    ) = [
        c_char_p,
        c_char_p,
    ], c_void_p
    lib.AddCustomServer.argtypes, lib.AddCustomServer.restype = [
        c_char_p,
        c_char_p,
    ], c_void_p
    lib.RemoveInstituteAccess.argtypes, lib.RemoveInstituteAccess.restype = [
        c_char_p,
        c_char_p,
    ], c_void_p
    lib.RemoveSecureInternet.argtypes, lib.RemoveSecureInternet.restype = [
        c_char_p
    ], c_void_p
    lib.RenewSession.argtypes, lib.RenewSession.restype = [c_char_p], c_void_p
    lib.SetConnected.argtypes, lib.SetConnected.restype = [c_char_p], c_void_p
    lib.SetConnecting.argtypes, lib.SetConnecting.restype = [c_char_p], c_void_p
    lib.Cleanup.argtypes, lib.Cleanup.restype = [
        c_char_p,
        cToken,
    ], c_void_p
    lib.SetDisconnected.argtypes, lib.SetDisconnected.restype = [c_char_p], c_void_p
    lib.SetDisconnecting.argtypes, lib.SetDisconnecting.restype = [c_char_p], c_void_p
    lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p, c_char_p], c_void_p
    lib.SetSearchServer.argtypes, lib.SetSearchServer.restype = [c_char_p], c_void_p
    lib.SetSecureLocation.argtypes, lib.SetSecureLocation.restype = [
        c_char_p,
        c_char_p,
    ], c_void_p
    lib.SetSupportWireguard.argtypes, lib.SetSupportWireguard.restype = [
        c_char_p,
        c_int,
    ], c_void_p
    lib.ShouldRenewButton.argtypes, lib.ShouldRenewButton.restype = [], int
    lib.StartFailover.argtypes, lib.StartFailover.restype = [
        c_char_p,
        c_char_p,
        c_int,
        ReadRxBytes,
    ], DataError
    lib.CancelFailover.argtypes, lib.CancelFailover.restype = [c_char_p], c_void_p
