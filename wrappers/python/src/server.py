from . import lib, cServers, cServerLocations
from ctypes import cast, POINTER


class Profile:
    def __init__(self, identifier, display_name, default_gateway: bool):
        self.identifier = identifier
        self.display_name = display_name
        self.default_gateway = default_gateway

    def __str__(self):
        return f"Profile: {self.display_name}"


class Server:
    def __init__(self, url, display_name, profiles, current_profile, expire_time):
        self.url = url
        self.display_name = display_name
        self.profiles = profiles
        self.current_profile = None
        if current_profile < len(profiles):
            self.current_profile = profiles[current_profile]
        self.expire_time = expire_time

    def __str__(self):
        return f"Server: {self.url}, with current profile: {self.current_profile}"


class InstituteServer(Server):
    def __init__(
        self, url, display_name, support_contact, profiles, current_profile, expire_time
    ):
        super().__init__(url, display_name, profiles, current_profile, expire_time)
        self.support_contact = support_contact

    def __str__(self):
        return f"Institute Server: {self.display_name}"


class SecureInternetServer(Server):
    def __init__(
        self,
        url,
        display_name,
        support_contact,
        profiles,
        current_profile,
        expire_time,
        country_code,
    ):
        super().__init__(url, display_name, profiles, current_profile, expire_time)
        self.support_contact = support_contact
        self.country_code = country_code

    def __str__(self):
        return f"Secure Internet Server: {self.display_name} with country {self.country_code}"


def get_type_for_str(type_str: str):
    if type_str is "secure_internet":
        return SecureInternetServer
    if type_str is "custom_server":
        return Server
    return InstituteServer


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
    profiles = []
    if not current_server.profiles:
        return None

    _profiles = current_server.profiles.contents
    current_profile = _profiles.current
    for i in range(_profiles.total_profiles):
        if not _profiles.profiles or not _profiles.profiles[i]:
            return None
        profile = _profiles.profiles[i].contents
        profiles.append(
            Profile(
                profile.identifier.decode("utf-8"),
                profile.display_name.decode("utf-8"),
                profile.default_gateway == 1,
            )
        )

    if _type is SecureInternetServer:
        return SecureInternetServer(
            identifier,
            display_name,
            support_contact,
            profiles,
            current_profile,
            current_server.expire_time,
            current_server.country_code.decode("utf-8"),
        )
    if _type is InstituteServer:
        return InstituteServer(
            identifier,
            display_name,
            support_contact,
            profiles,
            current_profile,
            current_server.expire_time,
        )
    return Server(
        identifier, display_name, profiles, current_profile, current_server.expire_time
    )


def get_servers(ptr):
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

def get_locations(ptr):
    if ptr:
        locations = cast(ptr, POINTER(cServerLocations)).contents
        location_list = []
        for i in range(locations.total_locations):
            location_list.append(locations.locations[i].decode("utf-8"))
        lib.FreeSecureLocations(ptr)
        return location_list
    return None
