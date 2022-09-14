import eduvpn_common.main as eduvpn
from eduvpn_common.state import State, StateType
import webbrowser
import json
import sys
import time
from typing import List

# Asks the user for a profile index
# It loops up until a valid input is given
def ask_profile_input(total: int) -> int:
    profile_index = None

    while profile_index is None:
        try:
            profile_index = int(
                input("Please select a profile by inputting a number (e.g. 1): ")
            )
            if (profile_index > total) or (profile_index < 1):
                print("Invalid profile range")
                profile_index = None
        except ValueError:
            print("Please enter a valid input")

    # The profile is one based, move to zero based input
    return profile_index - 1


# Sets up the callbacks using the provided class
def setup_callbacks(_eduvpn: eduvpn.EduVPN) -> None:
    # The callback that starst OAuth
    @_eduvpn.event.on(State.NO_SERVER, StateType.Enter)
    def no_server(old_state: str, servers) -> None:
        for server in servers:
            print(type(server))
            print(server)
    # It needs to open the URL in the web browser
    @_eduvpn.event.on(State.OAUTH_STARTED, StateType.Enter)
    def oauth_initialized(old_state: str, url: str) -> None:
        print(f"Got OAuth URL {url}, old state: {old_state}")
        webbrowser.open(url)

    @_eduvpn.event.on(State.ASK_LOCATION, StateType.Enter)
    def ask_location(old_state: str, locations: List[str]):
        _eduvpn.set_secure_location(locations[1])

    ## The callback which asks the user for a profile
    #@_eduvpn.event.on(State.ASK_PROFILE, StateType.Enter)
    #def ask_profile(old_state: str, profiles: str):
    #    print("Multiple profiles found, you need to select a profile:")

    #    # Parse the profiles as JSON
    #    data = json.loads(profiles)

    #    # Get a lits of profiles
    #    profile_strings = [x["profile_id"] for x in data["info"]["profile_list"]]
    #    total_profiles = len(profile_strings)

    #    # Create a list of the strings to standard output
    #    for idx, profile in enumerate(profile_strings):
    #        print(f"{idx+1}. {profile}")

    #    # Get the profile index from the user
    #    profile_index = ask_profile_input(total_profiles)

    #    # Set the profile with the index
    #    _eduvpn.set_profile(profile_strings[profile_index])


# The main entry point
if __name__ == "__main__":
    _eduvpn = eduvpn.EduVPN("org.eduvpn.app.linux", "configs")
    setup_callbacks(_eduvpn)

    # Register with the eduVPN-common library
    try:
        _eduvpn.register(debug=True)
    except Exception as e:
        print("Failed registering:", e)

    #server = input(
    #    "Which server (Custom/Institute Access) do you want to connect to? (e.g. https://eduvpn.example.com): "
    #)

    # Get a Wireguard/OpenVPN config
    try:
        config, config_type = _eduvpn.get_config_secure_internet("https://idp.geant.org")
        print(f"Got a config with type: {config_type} and contents:\n{config}")
    except Exception as e:
        print("Failed to connect:", e)
        # Save and exit
        _eduvpn.deregister()
        sys.exit(1)

    # Set the internal FSM state to connected
    try:
        _eduvpn.set_connecting()
        _eduvpn.set_connected()
    except Exception as e:
        print("Failed to set connected:", e)

    # Save and exit
    _eduvpn.deregister()
