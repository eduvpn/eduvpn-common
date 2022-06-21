from . import lib, VPNStateChange, GetDataError, GetMultipleDataError, GetPtrString
from ctypes import *
from enum import Enum
from typing import Callable, Optional, Tuple
from functools import wraps
import threading


class StateType(Enum):
    Enter = 1
    Leave = 2
    Wait = 3

EDUVPN_CALLBACK_PROPERTY = '_eduvpn_property_callback'

# A state transition decorator for classes
# To use this, make sure to register the class with `register_class_callbacks`
def class_state_transition(state: str, state_type: StateType) -> Callable:
    def wrapper(func):
        setattr(func, EDUVPN_CALLBACK_PROPERTY, (state, state_type))
        return func
    return wrapper

class EventHandler(object):
    def __init__(self):
        self.handlers = {}

    def remove_event(self, state: str, state_type: StateType, func: Callable):
        for key, values in self.handlers.copy().items():
            if key == (state, state_type):
                values.remove(func)
                if not values:
                    del self.handlers[key]
                else:
                    self.handlers[key] = values

    def add_event(self, state: str, state_type: StateType, func: Callable):
        if (state, state_type) not in self.handlers:
            self.handlers[(state, state_type)] = []
        self.handlers[(state, state_type)].append(func)

    # A decorator for standalone functions
    def on(self, state: str, state_type: StateType) -> Callable:
        def wrapped_f(func):
            self.add_event(state, state_type, func)
            return func

        return wrapped_f

    def run_state(
        self, state: str, other_state: str, state_type: StateType, data: str
    ) -> None:
        if (state, state_type) not in self.handlers:
            return
        for func in self.handlers[(state, state_type)]:
            func(other_state, data)

    def run(self, old_state: str, new_state: str, data: str) -> None:
        if old_state == new_state:
            return

        # First run leave transitions, then enter
        # The state is done when the wait event finishes
        self.run_state(old_state, new_state, StateType.Leave, data)
        self.run_state(new_state, old_state, StateType.Enter, data)
        self.run_state(new_state, old_state, StateType.Wait, data)


# Registers the python app with the Go code
# name: The name of the app to be registered
# state_callback: The callback to trigger whenever a state is changed
def Register(
    name: str, config_directory: str, state_callback: Optional[Callable], debug: bool
) -> str:
    if not state_callback:
        return "No callback provided"
    name_bytes = name.encode("utf-8")
    dir_bytes = config_directory.encode("utf-8")
    ptr_err = lib.Register(name_bytes, dir_bytes, state_callback, debug)
    err_string = GetPtrString(ptr_err)
    return err_string


def CancelOAuth(name: str) -> str:
    name_bytes = name.encode("utf-8")
    ptr_err = lib.CancelOAuth(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def Deregister(name: str) -> str:
    name_bytes = name.encode("utf-8")
    ptr_err = lib.Deregister(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def GetDiscoServers(name: str) -> Tuple[str, str]:
    name_bytes = name.encode("utf-8")
    servers, servers_err = GetDataError(lib.GetServersList(name_bytes))
    return servers, servers_err


def GetDiscoOrganizations(name: str) -> Tuple[str, str]:
    name_bytes = name.encode("utf-8")
    organizations, organizations_err = GetDataError(
        lib.GetOrganizationsList(name_bytes)
    )
    return organizations, organizations_err


def GetConnectConfig(
    name: str, url: str, is_secure_internet: bool, force_tcp: bool
) -> Tuple[str, str, str]:
    name_bytes = name.encode("utf-8")
    url_bytes = url.encode("utf-8")
    multiple_data_error = lib.GetConnectConfig(
        name_bytes, url_bytes, is_secure_internet, force_tcp
    )
    return GetMultipleDataError(multiple_data_error)


def SetConnected(name: str) -> str:
    name_bytes = name.encode("utf-8")
    ptr_err = lib.SetConnected(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def SetDisconnected(name: str) -> str:
    name_bytes = name.encode("utf-8")
    ptr_err = lib.SetDisconnected(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def SetSearchServer(name: str) -> str:
    name_bytes = name.encode("utf-8")
    ptr_err = lib.SetSearchServer(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def SetIdentifier(name: str, identifier: str) -> str:
    name_bytes = name.encode("utf-8")
    identifier_bytes = identifier.encode("utf-8")
    ptr_err = lib.SetIdentifier(name_bytes, identifier_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string


def GetIdentifier(name: str) -> Tuple[str, str]:
    name_bytes = name.encode("utf-8")
    identifier, identifier_err = GetDataError(lib.GetIdentifier(name_bytes))
    return identifier, identifier_err


# This has to be global as otherwise the callback is not alive
callback_function = None


def register_callback(eduvpn):
    global callback_function
    callback_function = VPNStateChange(
        lambda old_state, new_state, data: eduvpn.callback(
            old_state.decode(), new_state.decode(), data.decode()
        )
    )


def SetProfileID(name: str, profile_id: str) -> str:
    name_bytes = name.encode("utf-8")
    profile_bytes = profile_id.encode("utf-8")
    error_string = lib.SetProfileID(name_bytes, profile_bytes)
    return GetPtrString(error_string)


class EduVPN(object):
    def __init__(self, name: str, config_directory: str):
        self.event_handler = EventHandler()
        self.name = name
        self.config_directory = config_directory
        register_callback(self)

        # Callbacks that need to wait for specific events

        # The ask profile callback needs to wait for the UI thread to select a profile
        # This is stored in the profile_event
        self.profile_event: Optional[threading.Event] = None
        @self.event.on("Ask_Profile", StateType.Wait)
        def wait_profile_event(old_state: str, profiles: str):
            if self.profile_event:
                self.profile_event.wait()

    def cancel_oauth(self) -> None:
        cancel_oauth_err = CancelOAuth(self.name)

        if cancel_oauth_err:
            raise Exception(cancel_oauth_err)

    def deregister(self) -> None:
        deregister_err = Deregister(self.name)

        if deregister_err:
            raise Exception(deregister_err)

    def register(self, debug: bool = False) -> None:
        register_err = Register(
            self.name, self.config_directory, callback_function, debug
        )

        if register_err:
            raise Exception(register_err)

    def get_disco_servers(self) -> str:
        servers, servers_err = GetDiscoServers(self.name)

        if servers_err:
            raise Exception(servers_err)

        return servers

    def get_disco_organizations(self) -> str:
        organizations, organizations_err = GetDiscoOrganizations(self.name)

        if organizations_err:
            raise Exception(organizations_err)

        return organizations

    def get_config(self, url: str, is_secure_internet: bool = False, force_tcp: bool = False):
        # Because it could be the case that a profile callback is started, store a threading event
        # In the constructor, we have defined a wait event for Ask_Profile, this waits for this event to be set
        # The event is set in self.set_profile
        self.profile_event = threading.Event()
        config, config_type, config_err = GetConnectConfig(
            self.name, url, is_secure_internet, force_tcp
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
        connect_err = SetConnected(self.name)

        if connect_err:
            raise Exception(connect_err)

    def set_disconnected(self) -> None:
        disconnect_err = SetDisconnected(self.name)

        if disconnect_err:
            raise Exception(disconnect_err)

    def get_identifier(self) -> str:
        identifier, identifier_err = GetIdentifier(self.name)

        if identifier_err:
            raise Exception(identifier_err)

        return identifier

    def set_identifier(self, identifier: str) -> None:
        identifier_err = SetIdentifier(self.name, identifier)

        if identifier_err:
            raise Exception(identifier_err)

    def set_search_server(self) -> None:
        search_err = SetSearchServer(self.name)

        if search_err:
            raise Exception(search_err)

    def change_class_callbacks(self, cls, add=True) -> None:
        # Loop over method names
        for method_name in dir(cls):

            try:
                # Get the method
                method = getattr(cls, method_name)
            except:
                # Unable to get a value, go to the next
                continue

            # If it has a callback defined, add it to the events
            method_value = getattr(method, EDUVPN_CALLBACK_PROPERTY, None)
            if method_value:
                state, state_type = method_value

                if add:
                    self.event.add_event(state, state_type, method)
                else:
                    self.event.remove_event(state, state_type, method)

    def remove_class_callbacks(self, cls) -> None:
        self.change_class_callbacks(cls, add=False)

    def register_class_callbacks(self, cls) -> None:
        self.change_class_callbacks(cls)

    @property
    def event(self) -> EventHandler:
        return self.event_handler

    def callback(self, old_state: str, new_state: str, data: str) -> None:
        self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id: str) -> None:
        # Set the profile id
        profile_err = SetProfileID(self.name, profile_id)

        if profile_err:
            raise Exception(profile_err)

        # If there is a profile event, set it so that the wait callback finishes
        # And so that the Go code can move to the next state
        if self.profile_event:
            self.profile_event.set()
