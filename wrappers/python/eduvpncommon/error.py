from enum import Enum

class GoError(Exception):
    message_dict: dict
    code: Enum | None

    def __init__(self, err: Enum, messages: dict):
        assert err
        try:
            self.code = err
        except ValueError:
            self.code = None
        self.message_dict = messages

    def __str__(self):
        return self.message_dict[self.code] if self.code in self.message_dict else f"unknown error ({self.code})"
