from eduvpn_common.types import cServer, cServers, cServerLocations, cServerProfiles
from ctypes import cast, POINTER
from datetime import datetime


class Profile:
    def __init__(self, identifier, display_name, default_gateway: bool):
        self.identifier = identifier
        self.display_name = display_name
        self.default_gateway = default_gateway

    def __str__(self):
        return self.display_name


class Profiles:
    def __init__(self, profiles, current):
        self.profiles = profiles
        self.current_index = current

    @property
    def current(self):
        if self.current_index < len(self.profiles):
            return self.profiles[self.current_index]
        return None


class Server:
    def __init__(self, url, display_name, profiles=None, expire_time=0):
        self.url = url
        self.display_name = display_name
        self.profiles = profiles
        self.current_profile = None
        self.expire_time = datetime.fromtimestamp(expire_time)

    def __str__(self):
        return self.display_name

    @property
    def category(self):
        return "Custom Server"


class InstituteServer(Server):
    def __init__(self, url, display_name, support_contact, profiles, expire_time):
        super().__init__(url, display_name, profiles, expire_time)
        self.support_contact = support_contact

    @property
    def category(self):
        return "Institute Access Server"


class SecureInternetServer(Server):
    def __init__(
        self,
        org_id,
        display_name,
        support_contact,
        profiles,
        expire_time,
        country_code,
    ):
        super().__init__(org_id, display_name, profiles, expire_time)
        self.org_id = org_id
        self.support_contact = support_contact
        self.country_code = country_code

    @property
    def category(self):
        return "Secure Internet Server"


def get_type_for_str(type_str: str):
    if type_str == "secure_internet":
        return SecureInternetServer
    if type_str == "custom_server":
        return Server
    return InstituteServer


def get_profiles(ptr):
    if not ptr:
        return []
    profiles = []
    _profiles = ptr.contents
    current_profile = _profiles.current
    if not _profiles.profiles:
        return []
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


def get_server(ptr, _type=None):
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


def get_transition_server(lib, ptr):
    server = get_server(cast(ptr, POINTER(cServer)))
    lib.FreeServer(ptr)
    return server


def get_transition_profiles(lib, ptr):
    profiles = get_profiles(cast(ptr, POINTER(cServerProfiles)))
    lib.FreeProfiles(ptr)
    return profiles


def get_servers(lib, ptr):
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


def get_locations(lib, ptr):
    if ptr:
        locations = cast(ptr, POINTER(cServerLocations)).contents
        location_list = []
        for i in range(locations.total_locations):
            location_list.append(locations.locations[i].decode("utf-8"))
        lib.FreeSecureLocations(ptr)
        return location_list
    return None
