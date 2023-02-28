from ctypes import CDLL, POINTER, c_void_p, cast
from datetime import datetime
from typing import List, Optional, Type

from eduvpn_common.types import (
    cConfig,
    cServer,
    cServerLocations,
    cServerProfiles,
    cServers,
    cToken,
)


class Profile:
    """The class that represents a server profile.

    :param: identifier: str: The identifier (id) of the profile
    :param: display_name: str: The display name of the profile
    :param: default_gateway: str: Whether or not this profile should have the default gateway set
    """

    def __init__(self, identifier: str, display_name: str, default_gateway: bool):
        self.identifier = identifier
        self.display_name = display_name
        self.default_gateway = default_gateway

    def __str__(self):
        return self.display_name


class Token:
    """The class that represents oauth Tokens

    :param: access: str: The access token
    :param: refresh: str: The refresh token
    :param: expired: int: The expire unix time
    """

    def __init__(self, access: str, refresh: str, expired: int):
        self.access = access
        self.refresh = refresh
        self.expires = expired


class Config:
    """The class that represents an OpenVPN/WireGuard config

    :param: config: str: The config string
    :param: config_type: str: The type of config, openvpn/wireguard
    :param: tokens: Optional[Token]: The tokens
    """

    def __init__(self, config: str, config_type: str, tokens: Optional[Token]):
        self.config = config
        self.config_type = config_type
        self.tokens = tokens

    def __str__(self):
        return self.config


class Profiles:
    """The class that represents a list of profiles

    :param: profiles: List[Profile]: A list of profiles
    :param: current: int: The current profile index
    """

    def __init__(self, profiles: List[Profile], current: int):
        self.profiles = profiles
        self.current_index = current

    @property
    def current(self) -> Optional[Profile]:
        """Get the current profile if there is any

        :return: The profile if there is a current one (meaning the index is valid)
        :rtype: Optional[Profile]
        """
        if self.current_index < len(self.profiles):
            return self.profiles[self.current_index]
        return None


class Server:
    """The class that represents a server. Use this for a custom server

    :param: url: str: The base URL of the server. In case of secure internet (supertype) this is the organisation ID URL
    :param: display_name: str: The display name of the server
    :param: profiles: Optional[Profiles]: The profiles if there are any already obtained, defaults to None
    :param: expire_time: int: The expiry time in a Unix timestamp, defaults to 0
    """

    def __init__(
        self,
        url: str,
        display_name: str,
        profiles: Optional[Profiles] = None,
        expire_time: int = 0,
    ):
        self.url = url
        self.display_name = display_name
        self.profiles = profiles
        self.expire_time = datetime.fromtimestamp(expire_time)

    def __str__(self):
        return self.display_name

    @property
    def category(self) -> str:
        """Return the category of the server as a string

        :return: The category string, "Custom Server"
        :rtype: str
        """
        return "Custom Server"


class InstituteServer(Server):
    """The class that represents an Institute Access Server

    :param: url: str: The base URL of the Institute Access Server
    :param: display_name: str: The display name of the Institute Access Server
    :param: support_contact: List[str]: The list of support contacts
    :param: profiles: Profiles: The profiles of the server
    :param: expire_time: int: The expiry time in a Unix timestamp
    """

    def __init__(
        self,
        url: str,
        display_name: str,
        support_contact: List[str],
        profiles: Profiles,
        expire_time: int,
    ):
        super().__init__(url, display_name, profiles, expire_time)
        self.support_contact = support_contact

    @property
    def category(self) -> str:
        """Return the category of the institute server as a string

        :return: The category string, "Institute Access Server"
        :rtype: str
        """
        return "Institute Access Server"


class SecureInternetServer(Server):
    """The class that represents a Secure Internet Server

    :param: org_id: str: The organization ID of the Secure Internet Server as returned by Discovery
    :param: display_name: str: The display name of the server
    :param: support_contact: List[str]: The list of support contacts of the server
    :param: locations: List[str]: The list of secure internet locations
    :param: profiles: Profiles: The list of profiles that the server has
    :param: expire_time: int: The expiry time in a Unix timestamp
    :param: country_code: str: The country code of the server
    """

    def __init__(
        self,
        org_id: str,
        display_name: str,
        support_contact: List[str],
        locations: List[str],
        profiles: Profiles,
        expire_time: int,
        country_code: str,
    ):
        super().__init__(org_id, display_name, profiles, expire_time)
        self.org_id = org_id
        self.support_contact = support_contact
        self.locations = locations
        self.country_code = country_code

    @property
    def category(self) -> str:
        """Return the category of the secure internet server as a string

        :return: The category string, "Secure Internet Server"
        :rtype: str
        """
        return "Secure Internet Server"


def get_type_for_str(type_str: str) -> Type[Server]:
    """Get the right class type for a certain string input

    :param type_str: str: The string that represents the type of server, one of secure_internet, institute_access, custom_server

    :return: The server, defaults to Institute Server if an invalid input is given
    :rtype: Type[Server]
    """
    if type_str == "secure_internet":
        return SecureInternetServer
    if type_str == "custom_server":
        return Server
    return InstituteServer


def get_locations_from_ptr(ptr) -> List[str]:
    """Get the locations from the Go shared library and convert it to a Python usable structure

    :param ptr: The pointer to the List[str] locations as returned by the Go library

    :meta private:

    :return: Locations if there are any
    :rtype: List[str]
    """
    if not ptr:
        return []
    locations = cast(ptr, POINTER(cServerLocations)).contents
    location_list = []
    for i in range(locations.total_locations):
        location_list.append(locations.locations[i].decode("utf-8"))
    return location_list


def get_profiles(ptr) -> Optional[Profiles]:
    """Get the profiles from the Go shared library and convert it to a Python usable structure

    :param ptr: The pointer to the Profiles as returned by the Go library

    :meta private:

    :return: Profiles if there are any
    :rtype: Optional[Profiles]
    """
    if not ptr:
        return None
    profiles = []
    _profiles = ptr.contents
    current_profile = _profiles.current
    if not _profiles.profiles:
        return None
    for i in range(_profiles.total_profiles):
        if not _profiles.profiles[i]:
            continue
        profile = _profiles.profiles[i].contents
        profiles.append(
            Profile(
                profile.identifier.decode("utf-8"),
                profile.display_name.decode("utf-8"),
                profile.default_gateway == 1,
            )
        )
    return Profiles(profiles, current_profile)


def get_server(ptr, _type: Optional[str] = None) -> Optional[Server]:
    """Get the server from the Go shared library and convert it to a Python usable structure

    :param ptr: The pointer as returned by the Go library
    :param _type:  (Default value = None): The optional parameter that represents whether or not the type is enforced to the input. If None it is automatically determined

    :meta private:

    :return: Server if there is any
    :rtype: Optional[Server]
    """
    if not ptr:
        return None

    current_server = ptr.contents
    if _type is None:
        _type = get_type_for_str(current_server.server_type.decode("utf-8"))

    identifier = current_server.identifier.decode("utf-8")
    display_name = current_server.display_name.decode("utf-8")

    if _type is not Server:
        support_contact = []
        for i in range(current_server.total_support_contact):
            support_contact.append(current_server.support_contact[i].decode("utf-8"))
    locations = get_locations_from_ptr(current_server.locations)
    profiles = get_profiles(current_server.profiles)
    if profiles is None:
        profiles = Profiles([], 0)
    if _type is SecureInternetServer:
        return SecureInternetServer(
            identifier,
            display_name,
            support_contact,
            locations,
            profiles,
            current_server.expire_time,
            current_server.country_code.decode("utf-8"),
        )
    if _type is InstituteServer:
        return InstituteServer(
            identifier,
            display_name,
            support_contact,
            profiles,
            current_server.expire_time,
        )
    return Server(identifier, display_name, profiles, current_server.expire_time)


def get_transition_server(lib: CDLL, ptr: c_void_p) -> Optional[Server]:
    """Get a server from a transition event

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The Go's returned C pointer that represents the Server

    :meta private:

    :return: The server if there is any
    :rtype: Optional[Server]
    """
    if ptr:
        server = get_server(cast(ptr, POINTER(cServer)))
        lib.FreeServer(ptr)
        return server
    return None


def get_transition_profiles(lib: CDLL, ptr: c_void_p) -> Optional[Profiles]:
    """Get profiles from a transition event

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The Go's returned C pointer that represents the profiles

    :meta private:

    :return: The profiles if there is any
    :rtype: Optional[Profiles]
    """
    if ptr:
        profiles = get_profiles(cast(ptr, POINTER(cServerProfiles)))
        lib.FreeProfiles(ptr)
        return profiles
    return None


def get_servers(lib: CDLL, ptr: c_void_p) -> Optional[List[Server]]:
    """Get servers from the Go library as a C structure and return a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The C pointer to the servers structure

    :meta private:

    :return: The list of Servers if there is any
    :rtype: Optional[List[Server]]
    """
    if ptr:
        returned = []
        servers = cast(ptr, POINTER(cServers)).contents
        if servers.custom_servers:
            for i in range(servers.total_custom):
                current = get_server(servers.custom_servers[i], Server)
                if current is None:
                    continue
                returned.append(current)

        if servers.institute_servers:
            for i in range(servers.total_institute):
                current = get_server(servers.institute_servers[i], InstituteServer)
                if current is None:
                    continue
                returned.append(current)

        if servers.secure_internet:
            current = get_server(servers.secure_internet, SecureInternetServer)
            if current is not None:
                returned.append(current)
        lib.FreeServers(ptr)
        return returned
    return None


def get_locations(lib: CDLL, ptr: c_void_p) -> Optional[List[str]]:
    """Get locations from the Go library as a C structure and return a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The C pointer to the locations structure

    :meta private:

    :return: The list of servers if there are any
    :rtype: Optional[List[str]]
    """
    if ptr:
        location_list = get_locations_from_ptr(ptr)
        lib.FreeSecureLocations(ptr)
        return location_list
    return None


def get_config(lib: CDLL, ptr: c_void_p) -> Optional[Config]:
    """Get the config from the Go library as a C structure and return a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The C pointer to the confg structure

    :meta private:

    :return: The configuration if there is any
    :rtype: Optional[Config]
    """
    # TODO: FREE
    if ptr:
        config = cast(ptr, POINTER(cConfig)).contents
        cfg = config.config.decode("utf-8")
        cfg_type = config.config_type.decode("utf-8")
        tokens = None
        if config.token:
            token_struct = config.token.contents
            tokens = Token(
                token_struct.access.decode("utf-8"),
                token_struct.refresh.decode("utf-8"),
                token_struct.expired,
            )

        config_class = Config(cfg, cfg_type, tokens)
        lib.FreeConfig(ptr)
        return config_class
    return None


def encode_tokens(arg: Optional[Token]) -> cToken:
    if arg is None:
        return cToken("".encode("utf-8"), "".encode("utf-8"), 0)
    return cToken(arg.access.encode("utf-8"), arg.refresh.encode("utf-8"), arg.expires)
