from . import lib
from ctypes import *

def Register(name, url):
    name_bytes = name.encode('utf-8')
    url_bytes = url.encode('utf-8')
    lib.Register(name_bytes, url_bytes)

def InitializeOAuth():
    ptr = lib.InitializeOAuth()
    value = cast(ptr, c_char_p).value
    authURL = value.decode()
    lib.FreeString(ptr)
    return authURL
