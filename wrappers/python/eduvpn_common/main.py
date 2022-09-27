from typing import Optional, Tuple
import threading
from eduvpn_common.discovery import get_disco_organizations, get_disco_servers
from eduvpn_common.event import EventHandler
from eduvpn_common.loader import initialize_functions, load_lib
from eduvpn_common.types import VPNStateChange, encode_args, decode_res, get_data_error
from eduvpn_common.server import get_servers
from eduvpn_common.state import State, StateType

eduvpn_objects = {}


def add_as_global_object(eduvpn) -> bool:
    global eduvpn_objects
    if eduvpn.name not in eduvpn_objects:
        eduvpn_objects[eduvpn.name] = eduvpn
        return True
    return False


def remove_as_global_object(eduvpn):
    global eduvpn_objects
    eduvpn_objects.pop(eduvpn.name, None)


@VPNStateChange
def state_callback(name, old_state, new_state, data):
    name = name.decode()
    if name not in eduvpn_objects:
        return
    eduvpn_objects[name].callback(State(old_state), State(new_state), data)


class EduVPN(object):
    def __init__(self, name: str, config_directory: str):
        self.name = name
        self.config_directory = config_directory

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
        def wait_profile_event(old_state: int, profiles: str):
            if self.profile_event:
                self.profile_event.wait()

        @self.event.on(State.ASK_LOCATION, StateType.WAIT)
        def wait_location_event(old_state: int, locations: str):
            if self.location_event:
                self.location_event.wait()

    def go_function(self, func, *args):
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        return decode_res(func.restype)(self.lib, res)

    def go_function_custom_decode(self, func, decode_func, *args):
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        return decode_func(self.lib, res)

    def cancel_oauth(self) -> None:
        cancel_oauth_err = self.go_function(self.lib.CancelOAuth)

        if cancel_oauth_err:
            raise cancel_oauth_err

    def deregister(self) -> None:
        self.go_function(self.lib.Deregister)
        remove_as_global_object(self)

    def register(self, debug: bool = False) -> None:
        if not add_as_global_object(self):
            raise Exception("Already registered")

        register_err = self.go_function(
            self.lib.Register, self.config_directory, state_callback, debug
        )

        if register_err:
            raise register_err

    def get_disco_servers(self) -> str:
        servers, servers_err = self.go_function_custom_decode(
            self.lib.GetDiscoServers,
            decode_func=lambda lib, x: get_data_error(lib, x, get_disco_servers),
        )

        if servers_err:
            raise servers_err

        return servers

    def get_disco_organizations(self) -> str:
        organizations, organizations_err = self.go_function_custom_decode(
            self.lib.GetDiscoOrganizations,
            decode_func=lambda lib, x: get_data_error(lib, x, get_disco_organizations),
        )

        if organizations_err:
            raise organizations_err

        return organizations

    def remove_secure_internet(self):
        remove_err = self.go_function(self.lib.RemoveSecureInternet)

        if remove_err:
            raise remove_err

    def remove_institute_access(self, url: str):
        remove_err = self.go_function(self.lib.RemoveInstituteAccess, url)

        if remove_err:
            raise remove_err

    def remove_custom_server(self, url: str):
        remove_err = self.go_function(self.lib.RemoveCustomServer, url)

        if remove_err:
            raise remove_err

    def get_config(self, url: str, func: callable, force_tcp: bool = False):
        # Because it could be the case that a profile callback is started, store a threading event
        # In the constructor, we have defined a wait event for Ask_Profile, this waits for this event to be set
        # The event is set in self.set_profile
        self.profile_event = threading.Event()

        config, config_type, config_err = self.go_function(func, url, force_tcp)

        self.profile_event = None
        self.location_event = None

        if config_err:
            raise config_err

        return config, config_type

    def get_config_custom_server(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, self.lib.GetConfigCustomServer, force_tcp)

    def get_config_institute_access(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, self.lib.GetConfigInstituteAccess, force_tcp)

    def get_config_secure_internet(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        self.location_event = threading.Event()
        return self.get_config(url, self.lib.GetConfigSecureInternet, force_tcp)

    def go_back(self) -> None:
        # Ignore the error
        self.go_function(self.lib.GoBack)

    def set_connected(self) -> None:
        connect_err = self.go_function(self.lib.SetConnected)

        if connect_err:
            raise connect_err

    def set_disconnecting(self) -> None:
        disconnecting_err = self.go_function(self.lib.SetDisconnecting)

        if disconnecting_err:
            raise disconnecting_err

    def set_connecting(self) -> None:
        connecting_err = self.go_function(self.lib.SetConnecting)

        if connecting_err:
            raise connecting_err

    def set_disconnected(self, cleanup=True) -> None:
        disconnect_err = self.go_function(self.lib.SetDisconnected, cleanup)

        if disconnect_err:
            raise disconnect_err

    def set_search_server(self) -> None:
        search_err = self.go_function(self.lib.SetSearchServer)

        if search_err:
            raise search_err

    def remove_class_callbacks(self, cls) -> None:
        self.event_handler.change_class_callbacks(cls, add=False)

    def register_class_callbacks(self, cls) -> None:
        self.event_handler.change_class_callbacks(cls)

    @property
    def event(self) -> EventHandler:
        return self.event_handler

    def callback(self, old_state: State, new_state: State, data) -> None:
        self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id: str) -> None:
        # Set the profile id
        profile_err = self.go_function(self.lib.SetProfileID, profile_id)

        # If there is a profile event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.profile_event:
            self.profile_event.set()

        if profile_err:
            raise profile_err

    def change_secure_location(self) -> None:
        # Set the location by country code
        self.location_event = threading.Event()
        location_err = self.go_function(self.lib.ChangeSecureLocation)

        if location_err:
            raise location_err

    def set_secure_location(self, country_code: str) -> None:
        # Set the location by country code
        location_err = self.go_function(self.lib.SetSecureLocation, country_code)

        # If there is a location event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.location_event:
            self.location_event.set()

        if location_err:
            raise location_err

    def renew_session(self) -> None:
        renew_err = self.go_function(self.lib.RenewSession)

        if renew_err:
            raise renew_err

    def should_renew_button(self) -> bool:
        return self.go_function(self.lib.ShouldRenewButton)

    def in_fsm_state(self, state_id: State) -> bool:
        return self.go_function(self.lib.InFSMState, state_id)

    def get_saved_servers(self) -> str:
        return self.go_function_custom_decode(
            self.lib.GetSavedServers,
            decode_func=lambda lib, x: get_data_error(lib, x, get_servers),
        )
