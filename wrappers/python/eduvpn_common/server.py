from typing import List, Optional, Type
from eduvpn_common.types import cServer, cServers, cServerLocations, cServerProfiles
from ctypes import c_void_p, cast, POINTER, CDLL
from datetime import datetime


class Profile:
    def __init__(self, identifier: str, display_name: str, default_gateway: bool):
        self.identifier = identifier
        self.display_name = display_name
        self.default_gateway = default_gateway

    def __str__(self):
        return self.display_name


class Profiles:
    def __init__(self, profiles: List[Profile], current: int):
        self.profiles = profiles
        self.current_index = current

    @property
    def current(self) -> Optional[Profile]:
        if self.current_index < len(self.profiles):
            return self.profiles[self.current_index]
        return None


class Server:
    def __init__(self, url: str, display_name: str, profiles: Optional[Profiles] = None, expire_time: int = 0):
        self.url = url
        self.display_name = display_name
        self.profiles = profiles
        self.expire_time = datetime.fromtimestamp(expire_time)

    def __str__(self):
        return self.display_name

    @property
    def category(self) -> str:
        return "Custom Server"


class InstituteServer(Server):
    def __init__(self, url: str, display_name: str, support_contact: List[str], profiles: Profiles, expire_time: int):
        super().__init__(url, display_name, profiles, expire_time)
        self.support_contact = support_contact

    @property
    def category(self) -> str:
        return "Institute Access Server"

class SecureInternetServer(Server):
    def __init__(
        self,
        org_id: str,
        display_name: str,
        support_contact: List[str],
        profiles: Profiles,
        expire_time: int,
        country_code: str,
    ):
        super().__init__(org_id, display_name, profiles, expire_time)
        self.org_id = org_id
        self.support_contact = support_contact
        self.country_code = country_code

    @property
    def category(self) -> str:
        return "Secure Internet Server"


def get_type_for_str(type_str: str) -> Type[Server]:
    if type_str == "secure_internet":
        return SecureInternetServer
    if type_str == "custom_server":
        return Server
    return InstituteServer


def get_profiles(ptr) -> Optional[Profiles]:
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


def get_server(ptr, _type=None) -> Optional[Server]:
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
    profiles = get_profiles(current_server.profiles)
    if profiles is None:
        return None
    if _type is SecureInternetServer:
        return SecureInternetServer(
            identifier,
            display_name,
            support_contact,
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
    server = get_server(cast(ptr, POINTER(cServer)))
    lib.FreeServer(ptr)
    return server


def get_transition_profiles(lib: CDLL, ptr: c_void_p) -> Optional[Profiles]:
    profiles = get_profiles(cast(ptr, POINTER(cServerProfiles)))
    lib.FreeProfiles(ptr)
    return profiles


def get_servers(lib: CDLL, ptr: c_void_p) -> Optional[List[Server]]:
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
    if ptr:
        locations = cast(ptr, POINTER(cServerLocations)).contents
        location_list = []
        for i in range(locations.total_locations):
            location_list.append(locations.locations[i].decode("utf-8"))
        lib.FreeSecureLocations(ptr)
        return location_list
    return None
