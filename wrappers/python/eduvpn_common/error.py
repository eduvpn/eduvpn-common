from enum import Enum


class ErrorLevel(Enum):
    """The error level enum"""
    ERR_OTHER = 0
    ERR_INFO = 1
    ERR_WARNING = 2
    ERR_FATAL = 3


class WrappedError(Exception):
    """An exception returned by the Go library

    :param: traceback: str: The traceback of the error including newlines
    :param: cause: str: The cause of the error as a message
    :param: level: ErrorLevel: The level of the error
    """
    def __init__(self, traceback: str, cause: str, level: ErrorLevel):
        super(WrappedError, self).__init__(cause)
        self.traceback = traceback
        self.cause = cause
        self.level = level
