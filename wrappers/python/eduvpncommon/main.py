from . import lib, VPNStateChange
from ctypes import *

@VPNStateChange
def state_change(old, new, data):
    print(f"Python: State change {old.decode()} {new.decode()} DATA {data.decode()}")

# Registers the python app with the Go code
# name: The name of the app to be registered
# url: The url of the server to connect to, FIXME: To be removed
# state_callback: The callback to trigger whenever a state is changed, FIXME: Remove whenever this wrapper has implemented callbacks using function decorations
def Register(name, config_directory, state_callback):
    name_bytes = name.encode('utf-8')
    dir_bytes = config_directory.encode('utf-8')
    lib.Register(name_bytes, dir_bytes, state_callback)

