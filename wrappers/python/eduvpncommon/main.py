from . import lib, GOCB_StateChange
from ctypes import *

@GOCB_StateChange
def state_change(old, new):
    print(f"Python: State change {old.decode()} {new.decode()}")

def InitializeOAuth():
    ptr = lib.InitializeOAuth()
    value = cast(ptr, c_char_p).value
    authURL = value.decode()
    lib.FreeString(ptr)
    return authURL

# Registers the python app with the GO code
# name: The name of the app to be registered
# url: The url of the server to connect to, FIXME: To be removed
# state_callback: The callback to trigger whenever a state is changed, FIXME: Remove whenever this wrapper has implemented callbacks using function decorations
def Register(name, url, state_callback):
    name_bytes = name.encode('utf-8')
    url_bytes = url.encode('utf-8')
    lib.Register(name_bytes, url_bytes, state_callback)
