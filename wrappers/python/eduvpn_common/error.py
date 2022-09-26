from enum import Enum


class ErrorLevel(Enum):
    ERR_OTHER = 0
    ERR_INFO = 1
    ERR_WARNING = 2
    ERR_FATAL = 3


class WrappedError(Exception):
    def __init__(self, traceback: str, cause: str, level: ErrorLevel):
        super(WrappedError, self).__init__(cause)
        self.traceback = traceback
        self.cause = cause
        self.level = level
