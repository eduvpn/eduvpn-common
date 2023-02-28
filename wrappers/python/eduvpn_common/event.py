from ctypes import CDLL
from typing import Any, Callable, Dict, List, Tuple

from eduvpn_common.server import (
    get_locations,
    get_servers,
    get_transition_profiles,
    get_transition_server,
)
from eduvpn_common.state import State, StateType
from eduvpn_common.types import get_ptr_string

# The attribute that callback functions get
EDUVPN_CALLBACK_PROPERTY = "_eduvpn_property_callback"

# A state transition decorator for classes
# To use this, make sure to register the class with `register_class_callbacks`
def class_state_transition(state: int, state_type: StateType) -> Callable:
    """A decorator to be internally by classes to register the event

    :param state: int: The state of the transition
    :param state_type: StateType: The type of transition

    :meta private:
    """

    def wrapper(func):
        """

        :param func: The function to set the internal attribute for

        """
        setattr(func, EDUVPN_CALLBACK_PROPERTY, (state, state_type))
        return func

    return wrapper


def convert_data(lib: CDLL, state: int, data: Any) -> None:
    """The function that converts the C structure from the Go library to a Python structure

    :param lib: CDLL: The Go shared library
    :param state: int: The state to convert the data for
    :param data: Any: The data itself that has to be converted

    :meta private:
    """
    if not data:
        return None
    if state is State.NO_SERVER:
        return get_servers(lib, data)
    if state is State.OAUTH_STARTED:
        return get_ptr_string(lib, data)
    if state is State.ASK_LOCATION:
        return get_locations(lib, data)
    if state is State.ASK_PROFILE:
        return get_transition_profiles(lib, data)
    if state in [
        State.DISCONNECTED,
        State.DISCONNECTING,
        State.CONNECTING,
        State.CONNECTED,
    ]:
        return get_transition_server(lib, data)


class EventHandler(object):
    """The class that neatly handles event callbacks from the internal Go FSM"""

    def __init__(self, lib: CDLL):
        self.handlers: Dict[Tuple[int, StateType], List[Callable]] = {}
        self.lib = lib

    def change_class_callbacks(self, cls: Any, add: bool = True) -> None:
        """The function that is used to change class callbacks

        :param cls: Any: The class to change the callbacks for
        :param add: bool:  (Default value = True): Whether or not to add or remove the event. If true the event gets added

        :meta private:
        """
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

    def remove_event(self, state: int, state_type: StateType, func: Callable) -> None:
        """Removes an event

        :param state: int: The state to remove the event for
        :param state_type: StateType: The state type to remove the event for
        :param func: Callable: The function that needs to be removed from the event

        :meta private:
        """
        for key, values in self.handlers.copy().items():
            if key == (state, state_type):
                values.remove(func)
                if not values:
                    del self.handlers[key]
                else:
                    self.handlers[key] = values

    def add_event(self, state: int, state_type: StateType, func: Callable) -> None:
        """Adds an event

        :param state: int: The state to add the event for
        :param state_type: StateType: The state type to add the event for
        :param func: Callable: The function that needs to be added to the event

        :meta private:
        """
        if (state, state_type) not in self.handlers:
            self.handlers[(state, state_type)] = []
        self.handlers[(state, state_type)].append(func)

    def on(self, state: int, state_type: StateType) -> Callable:
        """The decorator for standalone functions

        :param state: int: The state of the event
        :param state_type: StateType: The state type of the event

        :meta private:
        """

        def wrapped_f(func):
            """

            :param func: The function to add the event for

            """
            self.add_event(state, state_type, func)
            return func

        return wrapped_f

    def run_state(
        self, state: int, other_state: int, state_type: StateType, data: str
    ) -> bool:
        """The function that runs the callback for a specific event

        :param state: int: The state of the event
        :param other_state: int: The other state of the event
        :param state_type: StateType: The state type of the event
        :param data: str: The data that gets passed to the function callback when the event is ran

        :meta private:
        """
        if (state, state_type) not in self.handlers:
            return False
        for func in self.handlers[(state, state_type)]:
            func(other_state, data)
        return True

    def run(
        self, old_state: int, new_state: int, data: Any, convert: bool = True
    ) -> bool:
        """Run a specific event.
        It converts the data and then runs the event for all state types

        :param old_state: int: The previous state for running the event
        :param new_state: int: The new state for running the event
        :param data: Any: The data that gets passed to the event
        :param convert: bool:  (Default value = True): Whether or not to convert the data further

        """
        # First run leave transitions, then enter
        # The state is done when the wait event finishes
        converted = data
        if convert:
            converted = convert_data(self.lib, new_state, data)
        self.run_state(old_state, new_state, StateType.LEAVE, converted)
        # We decide handled based on enter transitions
        handled = self.run_state(new_state, old_state, StateType.ENTER, converted)
        # Only run wait transitions if the enter transition is handled
        if handled:
            self.run_state(new_state, old_state, StateType.WAIT, converted)
        return handled
