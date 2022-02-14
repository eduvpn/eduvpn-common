from enum import Enum

class GoError(Exception):
    message_dict: dict
    code: Enum

    def __init__(self, err: Enum, messages: dict):
        assert err
        self.code = err
        self.message_dict = messages

    def __str__(self):
        return self.message_dict[self.code]
