from enum import IntEnum


class StateType(IntEnum):
    """
    The State Type enum.
    """

    ENTER = 1
    LEAVE = 2


StateEnum = IntEnum


class State(StateEnum):
    DEREGISTERED = 0
    MAIN = 1
    ADDING_SERVER = 2
    OAUTH_STARTED = 3
    GETTING_CONFIG = 4
    ASK_LOCATION = 5
    ASK_PROFILE = 6
    GOT_CONFIG = 7
    CONNECTING = 8
    CONNECTED = 9
    DISCONNECTING = 10
    DISCONNECTED = 11
