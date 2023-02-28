from enum import Enum


class WrappedError(Exception):
    """An exception returned by the Go library

    :param: traceback: str: The traceback of the error including newlines
    :param: cause: str: The cause of the error as a message
    """

    def __init__(self, traceback: str, cause: str):
        super(WrappedError, self).__init__(cause)
        self.traceback = traceback
        self.cause = cause
