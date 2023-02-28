import threading
from ctypes import cast, c_void_p, c_int, pointer
from typing import Any, Callable, Dict, Iterator, List, Optional, Tuple

from eduvpn_common.discovery import (
    DiscoOrganizations,
    DiscoServers,
    get_disco_organizations,
    get_disco_servers,
)
from eduvpn_common.event import EventHandler
from eduvpn_common.loader import initialize_functions, load_lib
from eduvpn_common.server import (
    Profiles,
    Config,
    Token,
    encode_tokens,
    get_config,
    Server,
    get_transition_server,
    get_servers,
)
from eduvpn_common.state import State, StateType
from eduvpn_common.types import (
    VPNStateChange,
    ReadRxBytes,
    cToken,
    decode_res,
    encode_args,
    get_data_error,
    get_bool,
)


class EduVPN(object):
    """The main class used to communicate with the Go library.
    It registers the client with the library and then calls the needed appropriate functions

    :param name: str: The name of the client. For commonly used names, see https://git.sr.ht/~fkooman/vpn-user-portal/tree/v3/item/src/OAuth/ClientDb.php. E.g. org.eduvpn.app.linux, if this name has "letsconnect" in it, then it is a Let's Connect! variant
    :param version: str: The version number of the client as a string, max 10 characters
    :param config_directory: str: The directory (absolute/relative) where to store the files
    :param language: str: The language of the client, e.g. en

    """

    def __init__(self, name: str, version: str, config_directory: str, language: str):
        self.name = name
        self.version = version
        self.config_directory = config_directory
        self.language = language

        # Load the library
        self.lib = load_lib()
        initialize_functions(self.lib)

        self.event_handler = EventHandler(self.lib)

        # Callbacks that need to wait for specific events

        # The ask profile callback needs to wait for the UI thread to select a profile
        # This is stored in the profile_event
        self.profile_event: Optional[threading.Event] = None
        self.location_event: Optional[threading.Event] = None

        @self.event.on(State.ASK_PROFILE, StateType.WAIT)
        def wait_profile_event(old_state: int, profiles: Profiles):
            """This functions waits until the ask location thread event is finished

            :param old_state: int: The old state of the profiles event
            :param profiles: Profiles: The profiles

            """
            if self.profile_event:
                self.profile_event.wait()

        @self.event.on(State.ASK_LOCATION, StateType.WAIT)
        def wait_location_event(old_state: int, locations: List[str]):
            """This functions waits until the location thread event is finished

            :param old_state: int: The old state of the location event
            :param locations: List[str]: The locations

            """
            if self.location_event:
                self.location_event.wait()

    def go_function(
        self, func: Any, *args: Iterator, decode_func: Optional[Callable] = None
    ) -> Any:
        """Call an internal go function and properly forward the arguments.
        Also handles decoding the result

        :param func: Any: The Go function to call from the shared library
        :param \*args: Iterator: The arguments to call the function with
        :param decode_func: Optional[Callable]:  (Default value = None): The function to decode the result into a Python type

        :meta private:
        """
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        if decode_func is None:
            return decode_res(func.restype)(self.lib, res)
        else:
            return decode_func(self.lib, res)

    def cancel_oauth(self) -> None:
        """Cancel the OAuth process"""
        cancel_oauth_err = self.go_function(self.lib.CancelOAuth)

        if cancel_oauth_err:
            raise cancel_oauth_err

    def deregister(self) -> None:
        """Deregister the Go shared library.
        This removes the object from internal bookkeeping and saves the configuration
        """
        self.go_function(self.lib.Deregister)
        remove_as_global_object(self)

    def register(self, debug: bool = False) -> None:
        """Register the Go shared library.
        This makes sure the FSM is initialized and that we can call Go functions

        :param debug: bool:  (Default value = False): Whether or not we want to enable debug logging

        """
        if not add_as_global_object(self):
            raise Exception("Already registered")

        register_err = self.go_function(
            self.lib.Register,
            self.version,
            self.config_directory,
            self.language,
            state_callback,
            debug,
        )

        if register_err:
            raise register_err

    def get_disco_servers(self) -> Optional[DiscoServers]:
        """Get the discovery servers

        :raises WrappedError: An error by the Go library

        :return: The disco Servers if any
        :rtype: Optional[DiscoServers]
        """
        servers, _ = self.go_function(
            self.lib.GetDiscoServers,
            decode_func=lambda lib, x: get_data_error(lib, x, get_disco_servers),
        )

        return servers

    def get_disco_organizations(self) -> Optional[DiscoOrganizations]:
        """Get the discovery organizations

        :raises WrappedError: An error by the Go library

        :return: The discovery Organizations if any
        :rtype: Optional[DiscoOrganizations]
        """
        organizations, _ = self.go_function(
            self.lib.GetDiscoOrganizations,
            decode_func=lambda lib, x: get_data_error(lib, x, get_disco_organizations),
        )

        return organizations

    def add_institute_access(self, url: str) -> None:
        """Add an institute access server

        :param url: str: The URL for the institute access server. Use the exact base_url as returned by Discovery

        :raises WrappedError: An error by the Go library
        """
        add_err = self.go_function(self.lib.AddInstituteAccess, url)

        if add_err:
            raise add_err

    def add_secure_internet_home(self, org_id: str) -> None:
        """Add a secure internet server

        :param org_id: str: The organization ID of the secure internet server. Use the exact organization as returned by Discovery

        :raises WrappedError: An error by the Go library
        """
        self.location_event = threading.Event()
        add_err = self.go_function(self.lib.AddSecureInternetHomeServer, org_id)

        if add_err:
            raise add_err

    def add_custom_server(self, url: str) -> None:
        """Add a custom server

        :param url: str: The base URL of the server

        :raises WrappedError: An error by the Go library
        """
        add_err = self.go_function(self.lib.AddCustomServer, url)

        if add_err:
            raise add_err

    def remove_secure_internet(self) -> None:
        """Remove the secure internet server

        :raises WrappedError: An error by the Go library
        """
        remove_err = self.go_function(self.lib.RemoveSecureInternet)

        if remove_err:
            raise remove_err

    def remove_institute_access(self, url: str) -> None:
        """Remove an institute access server

        :param url: str: The URL for the institute access server. Use the exact base_url as returned by Discovery

        :raises WrappedError: An error by the Go library
        """
        remove_err = self.go_function(self.lib.RemoveInstituteAccess, url)

        if remove_err:
            raise remove_err

    def remove_custom_server(self, url: str) -> None:
        """Remove a custom server

        :param url: str: The base URL of the server

        :raises WrappedError: An error by the Go library
        """
        remove_err = self.go_function(self.lib.RemoveCustomServer, url)

        if remove_err:
            raise remove_err

    def get_config(
        self,
        identifier: str,
        func: Any,
        prefer_tcp: bool = False,
        tokens: Optional[Token] = None,
    ) -> Optional[Config]:
        """Get an OpenVPN/WireGuard configuration from the server

        :param identifier: str: The identifier of the server, e.g. URL or ORG ID
        :param func: Any: The Go function to call
        :param prefer_tcp: bool:  (Default value = False): Whether or not to prefer TCP
        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :meta private:

        :raises WrappedError: An error by the Go library

        :return: The configuration and configuration type ('openvpn' or 'wireguard')
        :rtype: Config
        """
        # Because it could be the case that a profile callback is started, store a threading event
        # In the constructor, we have defined a wait event for Ask_Profile, this waits for this event to be set
        # The event is set in self.set_profile
        self.profile_event = threading.Event()

        config, config_err = self.go_function(
            func,
            identifier,
            prefer_tcp,
            encode_tokens(tokens),
            decode_func=lambda lib, x: get_data_error(lib, x, get_config),
        )

        self.profile_event = None
        self.location_event = None

        if config_err:
            raise config_err

        return config

    def get_config_custom_server(
        self, url: str, prefer_tcp: bool = False, tokens: Optional[Token] = None
    ) -> Optional[Config]:
        """Get an OpenVPN/WireGuard configuration from a custom server

        :param url: str: The URL of the custom server
        :param prefer_tcp: bool:  (Default value = False): Whether or not to prefer TCP
        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :raises WrappedError: An error by the Go library

        :return: The configuration and configuration type ('openvpn' or 'wireguard')
        :rtype: Config
        """
        return self.get_config(url, self.lib.GetConfigCustomServer, prefer_tcp, tokens)

    def get_config_institute_access(
        self, url: str, prefer_tcp: bool = False, tokens: Optional[Token] = None
    ) -> Optional[Config]:
        """Get an OpenVPN/WireGuard configuration from an institute access server

        :param url: str: The URL of the institute access server. Use the one from Discovery
        :param prefer_tcp: bool:  (Default value = False): Whether or not to prefer TCP
        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :raises WrappedError: An error by the Go library

        :return: The configuration and configuration type ('openvpn' or 'wireguard')
        :rtype: Config
        """
        return self.get_config(
            url, self.lib.GetConfigInstituteAccess, prefer_tcp, tokens
        )

    def get_config_secure_internet(
        self, org_id: str, prefer_tcp: bool = False, tokens: Optional[Token] = None
    ) -> Optional[Config]:
        """Get an OpenVPN/WireGuard configuration from a secure internet server

        :param org_id: str: The organization ID of the secure internet server. Use the one from Discovery
        :param prefer_tcp: bool:  (Default value = False): Whether or not to prefer TCP
        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :raises WrappedError: An error by the Go library

        :return: The configuration and configuration type ('openvpn' or 'wireguard')
        :rtype: Config
        """
        return self.get_config(
            org_id, self.lib.GetConfigSecureInternet, prefer_tcp, tokens
        )

    def go_back(self) -> None:
        """Go back in the FSM"""
        # Ignore the error
        self.go_function(self.lib.GoBack)

    def set_connected(self) -> None:
        """Set the FSM to connected

        :raises WrappedError: An error by the Go library
        """
        connect_err = self.go_function(self.lib.SetConnected)

        if connect_err:
            raise connect_err

    def set_disconnecting(self) -> None:
        """Set the FSM to disconnecting

        :raises WrappedError: An error by the Go library
        """
        disconnecting_err = self.go_function(self.lib.SetDisconnecting)

        if disconnecting_err:
            raise disconnecting_err

    def set_connecting(self) -> None:
        """Set the FSM to connecting

        :raises WrappedError: An error by the Go library
        """
        connecting_err = self.go_function(self.lib.SetConnecting)

        if connecting_err:
            raise connecting_err

    def cleanup(self, tokens: Optional[Token] = None) -> None:
        """Cleanup the vpn connection

        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :raises WrappedError: An error by the Go library
        """
        cleanup_err = self.go_function(self.lib.Cleanup, encode_tokens(tokens))

        if cleanup_err:
            raise cleanup_err

    def set_disconnected(
        self,
    ) -> None:
        """Set the FSM to disconnected

        :param cleanup: bool:  (Default value = True): Whether or not to call /disconnect to the server. This invalidates the OpenVPN/WireGuard configuration
        :param tokens: Optional[Token]  (Default value = None): The OAuth tokens if available

        :raises WrappedError: An error by the Go library
        """
        disconnect_err = self.go_function(self.lib.SetDisconnected)

        if disconnect_err:
            raise disconnect_err

    def set_search_server(self) -> None:
        """Set the FSM to search server

        :raises WrappedError: An error by the Go library
        """
        search_err = self.go_function(self.lib.SetSearchServer)

        if search_err:
            raise search_err

    def remove_class_callbacks(self, cls: Any) -> None:
        """Remove class callbacks

        :param cls: Any: The class to remove callbacks for

        """
        self.event_handler.change_class_callbacks(cls, add=False)

    def register_class_callbacks(self, cls: Any) -> None:
        """Register class callbacks

        :param cls: Any: The class to register callbacks for

        """
        self.event_handler.change_class_callbacks(cls)

    @property
    def event(self) -> EventHandler:
        """The property that gets the event handler

        :return: The event handler
        :rtype: EventHandler
        """
        return self.event_handler

    def callback(self, old_state: State, new_state: State, data: Any) -> bool:
        """Run an event callback

        :param old_state: State: The previous state
        :param new_state: State: The new state
        :param data: Any: The data to pass to the event

        """
        return self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id: str) -> None:
        """Set the profile of the current server

        :param profile_id: str: The profile id of the chosen profile for the server

        :raises WrappedError: An error by the Go library
        """
        # Set the profile id
        profile_err = self.go_function(self.lib.SetProfileID, profile_id)

        # If there is a profile event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.profile_event:
            self.profile_event.set()

        if profile_err:
            raise profile_err

    def change_secure_location(self) -> None:
        """Change the secure location. This calls the necessary events

        :raises WrappedError: An error by the Go library
        """
        # Set the location by country code
        self.location_event = threading.Event()
        location_err = self.go_function(self.lib.ChangeSecureLocation)

        if location_err:
            raise location_err

    def set_secure_location(self, country_code: str) -> None:
        """Set the secure location

        :param country_code: str: The country code of the new location

        :raises WrappedError: An error by the Go library
        """
        # Set the location by country code
        location_err = self.go_function(self.lib.SetSecureLocation, country_code)

        # If there is a location event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.location_event:
            self.location_event.set()

        if location_err:
            raise location_err

    def renew_session(self) -> None:
        """Renew the session. This invalidates the tokens and runs the necessary callbacks to log back in

        :raises WrappedError: An error by the Go library
        """
        renew_err = self.go_function(self.lib.RenewSession)

        if renew_err:
            raise renew_err

    def set_support_wireguard(self, support: bool) -> None:
        """Indicates whether or not the OS supports WireGuard connections.

        :param support: bool: whether or not wireguard is supported

        :raises WrappedError: An error by the Go library
        """
        support_err = self.go_function(self.lib.SetSupportWireguard, support)

        if support_err:
            raise support_err

    def should_renew_button(self) -> bool:
        """Whether or not the UI should show the renew button

        :return: Whether or not the return button should be shown
        :rtype: bool
        """
        return self.go_function(self.lib.ShouldRenewButton)

    def in_fsm_state(self, state_id: State) -> bool:
        """Check whether or not the FSM is in the provided state

        :param state_id: State: The state to check for

        :return: Whether or not the FSM is in the provided state
        :rtype: bool
        """
        return self.go_function(self.lib.InFSMState, state_id)

    def get_current_server(self) -> Optional[Server]:
        """Get the current server

        :return: The current servers if there is any
        :rtype: Optional[List[Servers]]
        """
        server, server_err = self.go_function(
            self.lib.GetCurrentServer,
            decode_func=lambda lib, x: get_data_error(lib, x, get_transition_server),
        )

        if server_err:
            raise server_err

        return server

    def get_saved_servers(self) -> Optional[List[Server]]:
        """Get a list of saved servers

        :return: The list of Servers if there are any
        :rtype: Optional[List[Servers]]
        """
        servers, servers_err = self.go_function(
            self.lib.GetSavedServers,
            decode_func=lambda lib, x: get_data_error(lib, x, get_servers),
        )

        if servers_err:
            raise servers_err

        return servers

    def start_failover(
        self, gateway: str, wg_mtu: int, readrxbytes: ReadRxBytes
    ) -> bool:
        dropped, dropped_err = self.go_function(
            self.lib.StartFailover,
            gateway,
            wg_mtu,
            readrxbytes,
            decode_func=lambda lib, x: get_data_error(lib, x, get_bool),
        )
        if dropped_err:
            raise dropped_err
        return dropped

    def cancel_failover(self):
        cancel_err = self.go_function(self.lib.CancelFailover)
        if cancel_err:
            raise cancel_err


eduvpn_objects: Dict[str, EduVPN] = {}


@VPNStateChange
def state_callback(name: bytes, old_state: int, new_state: int, data: Any) -> int:
    """The internal callback that is passed to the Go library

    :param name: bytes: The name of the client
    :param old_state: int: The old state
    :param new_state: int: The new state
    :param data: Any: The data that still needs to be converted

    :meta private:
    """
    name_decoded = name.decode()
    if name_decoded not in eduvpn_objects:
        return 0
    handled = eduvpn_objects[name_decoded].callback(
        State(old_state), State(new_state), data
    )
    if handled:
        return 1
    return 0


def add_as_global_object(eduvpn: EduVPN) -> bool:
    """Add the provided parameter to the global objects lists so we can call the callback

    :param eduvpn: EduVPN: The class to add

    :meta private:

    :return: Whether or not the object was added
    :rtype: bool
    """
    global eduvpn_objects
    if eduvpn.name not in eduvpn_objects:
        eduvpn_objects[eduvpn.name] = eduvpn
        return True
    return False


def remove_as_global_object(eduvpn: EduVPN) -> None:
    """Remove the provided parameter from the global objects list

    :param eduvpn: EduVPN: The class to remove

    :meta private:
    """
    global eduvpn_objects
    eduvpn_objects.pop(eduvpn.name, None)
