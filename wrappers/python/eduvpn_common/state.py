from enum import IntEnum


class StateType(IntEnum):
    """
    The State Type enum. Wait types are mostly used for internal code
    """

    ENTER = 1
    LEAVE = 2
    WAIT = 3


class State(IntEnum):
    """
    The State enum. Each state here also exists in the Go library
    """

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
    DISCONNECTED = 10
    DISCONNECTING = 11
    CONNECTING = 12
    CONNECTED = 13
