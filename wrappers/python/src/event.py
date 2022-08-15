from . import VPNStateChange
from enum import Enum
from typing import Callable
from .state import StateType


EDUVPN_CALLBACK_PROPERTY = "_eduvpn_property_callback"

# A state transition decorator for classes
# To use this, make sure to register the class with `register_class_callbacks`
def class_state_transition(state: int, state_type: StateType) -> Callable:
    def wrapper(func):
        setattr(func, EDUVPN_CALLBACK_PROPERTY, (state, state_type))
        return func

    return wrapper


class EventHandler(object):
    def __init__(self):
        self.handlers = {}

    def change_class_callbacks(self, cls, add=True) -> None:
        # Loop over method names
        for method_name in dir(cls):
            try:
                # Get the method
                method = getattr(cls, method_name)
            except:
                # Unable to get a value, go to the next
                continue

            # If it has a callback defined, add it to the events
            method_value = getattr(method, EDUVPN_CALLBACK_PROPERTY, None)
            if method_value:
                state, state_type = method_value

                if add:
                    self.add_event(state, state_type, method)
                else:
                    self.remove_event(state, state_type, method)

    def remove_event(self, state: int, state_type: StateType, func: Callable):
        for key, values in self.handlers.copy().items():
            if key == (state, state_type):
                values.remove(func)
                if not values:
                    del self.handlers[key]
                else:
                    self.handlers[key] = values

    def add_event(self, state: int, state_type: StateType, func: Callable):
        if (state, state_type) not in self.handlers:
            self.handlers[(state, state_type)] = []
        self.handlers[(state, state_type)].append(func)

    # A decorator for standalone functions
    def on(self, state: int, state_type: StateType) -> Callable:
        def wrapped_f(func):
            self.add_event(state, state_type, func)
            return func

        return wrapped_f

    def run_state(
        self, state: int, other_state: int, state_type: StateType, data: str
    ) -> None:
        if (state, state_type) not in self.handlers:
            return
        for func in self.handlers[(state, state_type)]:
            func(other_state, data)

    def run(self, old_state: int, new_state: int, data: str) -> None:
        if old_state == new_state:
            return

        # First run leave transitions, then enter
        # The state is done when the wait event finishes
        self.run_state(old_state, new_state, StateType.Leave, data)
        self.run_state(new_state, old_state, StateType.Enter, data)
        self.run_state(new_state, old_state, StateType.Wait, data)
