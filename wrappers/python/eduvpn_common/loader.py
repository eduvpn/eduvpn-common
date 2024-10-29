import pathlib
from ctypes import CDLL, POINTER, c_char_p, c_int, c_longlong, c_void_p, cdll

from eduvpn_common import __version__
from eduvpn_common.types import (
    BoolError,
    DataError,
    HandlerError,
    ProxySetup,
    ReadRxBytes,
    RefreshList,
    TokenGetter,
    TokenSetter,
    VPNStateChange,
)


def load_lib() -> CDLL:
    """The function that loads the Go shared library

    :meta private:

    :return: The Go shared library loaded with cdll.LoadLibrary from ctypes
    :rtype: CDLL
    """
    libfile = f"libeduvpn_common-{__version__}.so"

    lib = None

    # Try to load in the normal path
    try:
        lib = cdll.LoadLibrary(libfile)
        # Otherwise, library should have been copied to the lib/ folder
    except Exception:
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
    lib.Deregister.argtypes, lib.Deregister.restype = [], None
    lib.ExpiryTimes.argtypes, lib.ExpiryTimes.restype = [], DataError
    lib.FreeString.argtypes, lib.FreeString.restype = [c_void_p], None
    lib.DiscoOrganizations.argtypes, lib.DiscoOrganizations.restype = [c_int, c_char_p], DataError
    lib.DiscoServers.argtypes, lib.DiscoServers.restype = [c_int, c_char_p], DataError
    lib.GetConfig.argtypes, lib.GetConfig.restype = (
        [
            c_int,
            c_int,
            c_char_p,
            c_int,
            c_int,
        ],
        DataError,
    )
    lib.AddServer.argtypes, lib.AddServer.restype = (
        [
            c_int,
            c_int,
            c_char_p,
            POINTER(c_longlong),
        ],
        c_char_p,
    )
    lib.CurrentServer.argtypes, lib.CurrentServer.restype = [], DataError
    lib.RemoveServer.argtypes, lib.RemoveServer.restype = (
        [
            c_int,
            c_char_p,
        ],
        c_char_p,
    )
    lib.ServerList.argtypes, lib.ServerList.restype = [], DataError
    lib.Register.argtypes, lib.Register.restype = (
        [
            c_char_p,
            c_char_p,
            c_char_p,
            VPNStateChange,
            c_int,
        ],
        c_void_p,
    )
    lib.RenewSession.argtypes, lib.RenewSession.restype = [c_int], c_void_p
    lib.DiscoveryStartup.argtypes, lib.DiscoveryStartup.restype = [RefreshList], c_void_p
    lib.SetTokenHandler.argtypes, lib.SetTokenHandler.restype = (
        [
            TokenGetter,
            TokenSetter,
        ],
        c_void_p,
    )
    lib.CalculateGateway.argtypes, lib.CalculateGateway.restype = [c_char_p], DataError
    lib.Cleanup.argtypes, lib.Cleanup.restype = [c_int], c_void_p
    lib.SetProfileID.argtypes, lib.SetProfileID.restype = [c_char_p], c_void_p
    lib.CookieNew.argtypes, lib.CookieNew.restype = [], c_int
    lib.CookieReply.argtypes, lib.CookieReply.restype = [c_int, c_char_p], c_void_p
    lib.CookieCancel.argtypes, lib.CookieCancel.restype = [c_int], c_void_p
    lib.CookieDelete.argtypes, lib.CookieDelete.restype = [c_int], c_void_p
    lib.SetSecureLocation.argtypes, lib.SetSecureLocation.restype = (
        [
            c_char_p,
            c_char_p,
        ],
        c_void_p,
    )
    lib.SetState.argtypes, lib.SetState.restype = (
        [
            c_int,
        ],
        c_void_p,
    )
    lib.InState.argtypes, lib.InState.restype = (
        [
            c_int,
        ],
        BoolError,
    )
    lib.StartFailover.argtypes, lib.StartFailover.restype = (
        [
            c_int,
            c_char_p,
            c_int,
            ReadRxBytes,
        ],
        BoolError,
    )
    lib.NewProxyguard.argtypes, lib.NewProxyguard.restype = (
        [
            c_int,
            c_int,
            c_int,
            c_char_p,
            ProxySetup,
        ],
        HandlerError,
    )
    lib.ProxyguardTunnel.argtypes, lib.ProxyguardTunnel.restype = (
        [
            c_int,
            c_int,
            c_int,
        ],
        c_char_p,
    )
    lib.ProxyguardPeerIPs.argtypes, lib.ProxyguardPeerIPs.restype = (
        [
            c_int,
        ],
        DataError,
    )
