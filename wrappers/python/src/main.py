from . import lib, VPNStateChange, encode_args, decode_res
from typing import Optional, Tuple
import threading
from .event import StateType, EventHandler

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
    eduvpn_objects[name].callback(old_state.decode(), new_state.decode(), data.decode())


class EduVPN(object):
    def __init__(self, name: str, config_directory: str):
        self.event_handler = EventHandler()
        self.name = name
        self.config_directory = config_directory

        # Callbacks that need to wait for specific events

        # The ask profile callback needs to wait for the UI thread to select a profile
        # This is stored in the profile_event
        self.profile_event: Optional[threading.Event] = None

        @self.event.on("Ask_Profile", StateType.Wait)
        def wait_profile_event(old_state: str, profiles: str):
            if self.profile_event:
                self.profile_event.wait()

    def go_function(self, func, *args):
        # The functions all have at least one arg type which is the name of the client
        args_gen = encode_args(list(args), func.argtypes[1:])
        res = func(self.name.encode("utf-8"), *(args_gen))
        return decode_res(func.restype)(res)

    def cancel_oauth(self) -> None:
        cancel_oauth_err = self.go_function(lib.CancelOAuth)

        if cancel_oauth_err:
            raise Exception(cancel_oauth_err)

    def deregister(self) -> None:
        deregister_err = self.go_function(lib.Deregister)

        remove_as_global_object(self)
        if deregister_err:
            raise Exception(deregister_err)

    def register(self, debug: bool = False) -> None:
        if not add_as_global_object(self):
            raise Exception("Already registered")

        register_err = self.go_function(
            lib.Register, self.config_directory, state_callback, debug
        )

        if register_err:
            raise Exception(register_err)

    def get_disco_servers(self) -> str:
        servers, servers_err = self.go_function(lib.GetDiscoServers)

        if servers_err:
            raise Exception(servers_err)

        return servers

    def get_disco_organizations(self) -> str:
        organizations, organizations_err = self.go_function(lib.GetDiscoOrganizations)

        if organizations_err:
            raise Exception(organizations_err)

        return organizations

    def get_config(
        self, url: str, is_secure_internet: bool = False, force_tcp: bool = False
    ):
        # Because it could be the case that a profile callback is started, store a threading event
        # In the constructor, we have defined a wait event for Ask_Profile, this waits for this event to be set
        # The event is set in self.set_profile
        self.profile_event = threading.Event()
        config, config_type, config_err = self.go_function(
            lib.GetConnectConfig, url, is_secure_internet, force_tcp
        )

        if config_err:
            raise Exception(config_err)

        self.profile_event = None

        return config, config_type

    def get_config_institute_access(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, False, force_tcp)

    def get_config_secure_internet(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        return self.get_config(url, True, force_tcp)

    def set_connected(self) -> None:
        connect_err = self.go_function(lib.SetConnected)

        if connect_err:
            raise Exception(connect_err)

    def set_disconnected(self) -> None:
        disconnect_err = self.go_function(lib.SetDisconnected)

        if disconnect_err:
            raise Exception(disconnect_err)

    def get_identifier(self) -> str:
        identifier, identifier_err = self.go_function(lib.GetIdentifier)

        if identifier_err:
            raise Exception(identifier_err)

        return identifier

    def set_identifier(self, identifier: str) -> None:
        identifier_err = self.go_function(lib.SetIdentifier, identifier)

        if identifier_err:
            raise Exception(identifier_err)

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

    def callback(self, old_state: str, new_state: str, data: str) -> None:
        self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id: str) -> None:
        # Set the profile id
        profile_err = self.go_function(lib.SetProfileID, profile_id)

        if profile_err:
            raise Exception(profile_err)

        # If there is a profile event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.profile_event:
            self.profile_event.set()
