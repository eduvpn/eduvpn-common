from . import lib, VPNStateChange, GetDataError, GetPtrString
from ctypes import *
from enum import Enum
import functools

class StateType(Enum):
    Enter = 1
    Leave = 2

# Registers the python app with the Go code
# name: The name of the app to be registered
# url: The url of the server to connect to, FIXME: To be removed
# state_callback: The callback to trigger whenever a state is changed, FIXME: Remove whenever this wrapper has implemented callbacks using function decorations
def Register(name, config_directory, state_callback):
    name_bytes = name.encode("utf-8")
    dir_bytes = config_directory.encode("utf-8")
    ptr_err = lib.Register(name_bytes, dir_bytes, state_callback)
    err_string = GetPtrString(ptr_err)
    return err_string


class EduVPN(object):
    def __init__(self, name, config_directory):
        self.event_handler = EventHandler()
        self.name = name
        self.config_directory = config_directory

    def register(self) -> bool:
        closure = VPNStateChange(
            lambda old_state, new_state, data: self.callback(
                old_state.decode(), new_state.decode(), data.decode()
            )
        )
        return Register(self.name, self.config_directory, closure) == ""

    @property
    def event(self):
        return self.event_handler

    def callback(self, old_state, new_state, data):
        self.event.run(old_state, new_state, data)


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


def GetDiscoServers():
    servers, serversErr = GetDataError(lib.GetServersList())
    organizations, organizationsErr = GetDataError(lib.GetOrganizationsList())
    return servers, serversErr, organizations, organizationsErr
