from . import lib, VPNStateChange, GetDataError, GetPtrString
from ctypes import *
from enum import Enum


class StateType(Enum):
    Enter = 1
    Leave = 2


# Registers the python app with the Go code
# name: The name of the app to be registered
# state_callback: The callback to trigger whenever a state is changed
def Register(name, config_directory, state_callback, debug):
    name_bytes = name.encode("utf-8")
    dir_bytes = config_directory.encode("utf-8")
    ptr_err = lib.Register(name_bytes, dir_bytes, state_callback, debug)
    err_string = GetPtrString(ptr_err)
    return err_string

def CancelOAuth(name):
    name_bytes = name.encode("utf-8")
    ptr_err = lib.CancelOAuth(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string

def Deregister(name):
    name_bytes = name.encode("utf-8")
    ptr_err = lib.Deregister(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string

def GetDiscoServers(name):
    name_bytes = name.encode("utf-8")
    servers, serversErr = GetDataError(lib.GetServersList(name_bytes))
    organizations, organizationsErr = GetDataError(lib.GetOrganizationsList(name_bytes))
    return servers, serversErr, organizations, organizationsErr

def GetConnectConfig(name, url, is_secure_internet, force_tcp):
    name_bytes = name.encode("utf-8")
    url_bytes = url.encode("utf-8")
    data_error = lib.GetConnectConfig(name_bytes, url_bytes, is_secure_internet, force_tcp)
    return GetDataError(data_error)

def SetConnected(name):
    name_bytes = name.encode("utf-8")
    ptr_err = lib.SetConnected(name_bytes)
    err_string = GetPtrString(ptr_err)
    return err_string

def SetDisconnected(name):
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


def SetProfileID(name, profile_id) -> str:
    name_bytes = name.encode("utf-8")
    profile_bytes = profile_id.encode("utf-8")
    error_string = lib.SetProfileID(name_bytes, profile_bytes)
    return GetPtrString(error_string)


class EduVPN(object):
    def __init__(self, name, config_directory):
        self.event_handler = EventHandler()
        self.name = name
        self.config_directory = config_directory
        register_callback(self)

    def cancel_oauth(self) -> str:
        return CancelOAuth(self.name)

    def deregister(self) -> str:
        return Deregister(self.name)

    def register(self, debug=False) -> bool:
        return Register(self.name, self.config_directory, callback_function, debug) == ""

    def get_disco(self):
        return GetDiscoServers(self.name)

    def get_config_institute_access(self, url, force_tcp=False):
        return GetConnectConfig(self.name, url, False, force_tcp)

    def get_config_secure_internet(self, url, force_tcp=False):
        return GetConnectConfig(self.name, url, True, force_tcp)

    def set_disconnected(self):
        return SetDisconnected(self.name)

    def set_connected(self):
        return SetConnected(self.name)

    @property
    def event(self):
        return self.event_handler

    def callback(self, old_state, new_state, data):
        self.event.run(old_state, new_state, data)

    def set_profile(self, profile_id) -> str:
        return SetProfileID(self.name, profile_id)


class EventHandler(object):
    def __init__(self):
        self.handlers = {}

    def on(self, state, state_type):
        def wrapped_f(func):
            if (state, state_type) not in self.handlers:
                self.handlers[(state, state_type)] = []
            self.handlers[(state, state_type)].append(func)
            return func

        return wrapped_f

    def run_state(self, state, state_type, data):
        if (state, state_type) not in self.handlers:
            return
        for func in self.handlers[(state, state_type)]:
            func(data)

    def run(self, old_state, new_state, data):
        if old_state == new_state:
            return
        self.run_state(old_state, StateType.Leave, data)
        self.run_state(new_state, StateType.Enter, data)
