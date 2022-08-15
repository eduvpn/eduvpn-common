from enum import IntEnum


class StateType(IntEnum):
    Enter = 1
    Leave = 2
    Wait = 3


class State(IntEnum):
    DEREGISTERED = 0
    NO_SERVER = 1
    ASK_LOCATION = 2
    SEARCH_SERVER = 3
    LOADING_SERVER = 4
    CHOSEN_SERVER = 5
    OAUTH_STARTED = 6
    AUTHORIZED = 7
    REQUEST_CONFIG = 8
    ASK_PROFILE = 9
    HAS_CONFIG = 10
    CONNECTING = 11
    CONNECTED = 12
