from . import lib, VPNStateChange, encode_args, decode_res
from typing import Optional, Tuple
import threading
from .discovery import get_disco_organizations, get_disco_servers
from .event import EventHandler
from .state import State, StateType
from .server import get_servers
import json

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
        self.event_handler = EventHandler()
        self.name = name
        self.config_directory = config_directory

        # Callbacks that need to wait for specific events

        # The ask profile callback needs to wait for the UI thread to select a profile
        # This is stored in the profile_event
        self.profile_event: Optional[threading.Event] = None
        self.location_event: Optional[threading.Event] = None

        @self.event.on(State.ASK_PROFILE, StateType.Wait)
        def wait_profile_event(old_state: int, profiles: str):
            if self.profile_event:
                self.profile_event.wait()

        @self.event.on(State.ASK_LOCATION, StateType.Wait)
        def wait_location_event(old_state: int, locations: str):
            if self.location_event:
                self.location_event.wait()

    def go_function(self, func, *args):
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        return decode_res(func.restype)(res)

    def go_function_custom_decode(self, func, decode_func, *args):
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        return decode_func(res)

    def cancel_oauth(self) -> None:
        cancel_oauth_err = self.go_function(lib.CancelOAuth)

        if cancel_oauth_err:
            raise Exception(cancel_oauth_err)

    def deregister(self) -> None:
        self.go_function(lib.Deregister)
        remove_as_global_object(self)

    def register(self, debug: bool = False) -> None:
        if not add_as_global_object(self):
            raise Exception("Already registered")

        register_err = self.go_function(
            lib.Register, self.config_directory, state_callback, debug
        )

        if register_err:
            raise Exception(register_err)

    def get_disco_servers(self) -> str:
        servers = self.go_function_custom_decode(
            lib.GetDiscoServers, decode_func=get_disco_servers
        )

        # if servers_err:
        #    raise Exception(servers_err)

        return servers

    def get_disco_organizations(self) -> str:
        organizations = self.go_function_custom_decode(
            lib.GetDiscoOrganizations, decode_func=get_disco_organizations
        )
        # if organizations_err:
        #    raise Exception(organizations_err)

        return organizations

    def remove_secure_internet(self):
        remove_err = self.go_function(lib.RemoveSecureInternet)

        if remove_err:
            raise Exception(remove_err)

    def remove_institute_access(self, url: str):
        remove_err = self.go_function(lib.RemoveInstituteAccess, url)

        if remove_err:
            raise Exception(remove_err)

    def remove_custom_server(self, url: str):
        remove_err = self.go_function(lib.RemoveCustomServer, url)

        if remove_err:
            raise Exception(remove_err)

    def get_config(self, url: str, func: callable, force_tcp: bool = False):
        # Because it could be the case that a profile callback is started, store a threading event
        # In the constructor, we have defined a wait event for Ask_Profile, this waits for this event to be set
        # The event is set in self.set_profile
        self.profile_event = threading.Event()

        config_json, config_err = self.go_function(func, url, force_tcp)

        self.profile_event = None
        self.location_event = None

        if config_err:
            raise Exception(config_err)

        config_json_dict = json.loads(config_json)
        config = config_json_dict["config"]
        config_type = config_json_dict["config_type"]

        return config, config_type

    def get_config_custom_server(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, lib.GetConfigCustomServer, force_tcp)

    def get_config_institute_access(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, lib.GetConfigInstituteAccess, force_tcp)

    def get_config_secure_internet(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        self.location_event = threading.Event()
        return self.get_config(url, lib.GetConfigSecureInternet, force_tcp)

    def go_back(self) -> None:
        # Ignore the error
        self.go_function(lib.GoBack)

    def set_connected(self) -> None:
        connect_err = self.go_function(lib.SetConnected)

        if connect_err:
            raise Exception(connect_err)

    def set_disconnecting(self) -> None:
        disconnecting_err = self.go_function(lib.SetDisconnecting)

        if disconnecting_err:
            raise Exception(disconnecting_err)

    def set_connecting(self) -> None:
        connecting_err = self.go_function(lib.SetConnecting)

        if connecting_err:
            raise Exception(connecting_err)

    def set_disconnected(self, cleanup=True) -> None:
        disconnect_err = self.go_function(lib.SetDisconnected, cleanup)

        if disconnect_err:
            raise Exception(disconnect_err)

    def set_search_server(self) -> None:
        search_err = self.go_function(lib.SetSearchServer)

        if search_err:
            raise Exception(search_err)

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
        profile_err = self.go_function(lib.SetProfileID, profile_id)

        # If there is a profile event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.profile_event:
            self.profile_event.set()

        if profile_err:
            raise Exception(profile_err)

    def change_secure_location(self) -> None:
        # Set the location by country code
        self.location_event = threading.Event()
        location_err = self.go_function(lib.ChangeSecureLocation)

        if location_err:
            raise Exception(location_err)

    def set_secure_location(self, country_code: str) -> None:
        # Set the location by country code
        location_err = self.go_function(lib.SetSecureLocation, country_code)

        # If there is a location event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.location_event:
            self.location_event.set()

        if location_err:
            raise Exception(location_err)

    def renew_session(self) -> None:
        renew_err = self.go_function(lib.RenewSession)

        if renew_err:
            raise Exception(renew_err)

    def should_renew_button(self) -> bool:
        return self.go_function(lib.ShouldRenewButton)

    def in_fsm_state(self, state_id: State) -> bool:
        return self.go_function(lib.InFSMState, state_id)

    def get_saved_servers_old(self) -> str:
        return self.go_function(lib.GetSavedServersOLD)

    def get_saved_servers_new(self) -> str:
        return self.go_function_custom_decode(
            lib.GetSavedServersNEW, decode_func=get_servers
        )
