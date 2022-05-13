from . import lib, VPNStateChange, GetDataError, GetMultipleDataError, GetPtrString
from ctypes import *
from enum import Enum
from typing import Callable, Optional, Tuple


class StateType(Enum):
    Enter = 1
    Leave = 2


class EventHandler(object):
    def __init__(self):
        self.handlers = {}

    def on(self, state: str, state_type: StateType) -> Callable:
        def wrapped_f(func):
            if (state, state_type) not in self.handlers:
                self.handlers[(state, state_type)] = []
            self.handlers[(state, state_type)].append(func)
            return func

        return wrapped_f

    def run_state(self, state: str, other_state: str, state_type: StateType, data: str) -> None:
        if (state, state_type) not in self.handlers:
            return
        for func in self.handlers[(state, state_type)]:
            func(other_state, data)

    def run(self, old_state: str, new_state: str, data: str) -> None:
        if old_state == new_state:
            return
        self.run_state(old_state, new_state, StateType.Leave, data)
        self.run_state(new_state, old_state, StateType.Enter, data)


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

    def get_config_institute_access(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        config, config_type, config_err = GetConnectConfig(
            self.name, url, False, force_tcp
        )

        if config_err:
            raise Exception(config_err)

        return config, config_type

    def get_config_secure_internet(
        self, url: str, force_tcp: bool = False
    ) -> Tuple[str, str]:
        config, config_type, config_err = GetConnectConfig(
            self.name, url, True, force_tcp
        )

        if config_err:
            raise Exception(config_err)

        return config, config_type

    def set_disconnected(self) -> None:
        disconnect_err = SetDisconnected(self.name)

        if disconnect_err:
            raise Exception(disconnect_err)

    def set_connected(self) -> None:
        connect_err = SetConnected(self.name)

        if connect_err:
            raise Exception(connect_err)

    @property
    def event(self) -> EventHandler:
        return self.event_handler

    def callback(self, old_state: str, new_state: str, data: str) -> None:
        self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id: str) -> None:
        profile_err = SetProfileID(self.name, profile_id)

        if profile_err:
            raise Exception(profile_err)
