from ctypes import CDLL, POINTER, c_void_p, cast
from typing import List, Optional

from eduvpn_common.types import (
    cDiscoveryOrganizations,
    cDiscoveryServers,
    get_ptr_list_strings,
)


class DiscoOrganization:
    """The class that represents an organization from discovery

    :param: display_name: str: The display name of the organizations
    :param: org_id: str: The organization ID
    :param: secure_internet_home: str: Indicating which server is the secure internet home server
    :param: keyword_list: The list of strings that the users gets to search on to find the server
    """

    def __init__(
        self,
        display_name: str,
        org_id: str,
        secure_internet_home: str,
        keyword_list: List[str],
    ):
        self.display_name = display_name
        self.org_id = org_id
        self.secure_internet_home = secure_internet_home
        self.keyword_list = keyword_list

    def __str__(self):
        return self.display_name


class DiscoOrganizations:
    """The class that represents the list of disco organizations from discovery.
    Additionally it has provided a version which indicates which exact 'version' was used from discovery

    :param: version: int: The version of the list as returned by Discovery
    :param: organizations: List[DiscoOrganization]: The actual list of discovery organizations
    """

    def __init__(self, version: int, organizations: List[DiscoOrganization]):
        self.version = version
        self.organizations = organizations


class DiscoServer:
    """The class that represents a discovery server, this can be an institute access or secure internet server

    :param: authentication_url_template: str: The OAuth template to use to skip WAYF
    :param: base_url: str: The base URL of the server
    :param: country_code: str: The country code of the server
    :param: display_name: str: The display name of the server
    :param: keyword_list: List[str]: The list of keywords that the user can use to find the server
    :param: public_keys: List[str]: The list of public keys
    :param: server_type: str: The server type as a string
    :param: support_contacts: List[str]: The list of support contacts
    """

    def __init__(
        self,
        authentication_url_template: str,
        base_url: str,
        country_code: str,
        display_name: str,
        keyword_list: List[str],
        public_keys: List[str],
        server_type: str,
        support_contacts: List[str],
    ):
        self.authentication_url_template = authentication_url_template
        self.base_url = base_url
        self.country_code = country_code
        self.display_name = display_name
        self.keyword_list = keyword_list
        self.public_keys = public_keys
        self.server_type = server_type
        self.support_contacts = support_contacts

    def __str__(self):
        return self.display_name


class DiscoServers:
    """This class represents the list of discovery servers.
    The version indicates which exact 'version' from Discovery was used.

    :param: version: int: The version of the list as returned by Discovery
    :param: servers: List[DiscoServers]: The list of discovery servers
    """

    def __init__(self, version: int, servers: List[DiscoServer]):
        self.version = version
        self.servers = servers


def get_disco_organization(ptr) -> Optional[DiscoOrganization]:
    """Gets a discovery organization from the Go library in a C structure and returns a Python usable structure

    :param ptr: The pointer returned by the go library that contains a discovery organization

    :meta private:

    :return: The Discovery Organization if there is one
    :rtype: Optional[DiscoOrganization]
    """
    if not ptr:
        return None

    current_organization = ptr.contents
    display_name = current_organization.display_name.decode("utf-8")
    org_id = current_organization.org_id.decode("utf-8")
    secure_internet_home = current_organization.secure_internet_home.decode("utf-8")
    keyword_list = current_organization.keyword_list.decode("utf-8")
    return DiscoOrganization(display_name, org_id, secure_internet_home, keyword_list)


def get_disco_server(lib: CDLL, ptr) -> Optional[DiscoServer]:
    """Gets a discovery server from the Go library in a C structure and returns a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: The pointer to a discovery server returned by the Go library

    :meta private:

    :return: The Discovery Server if there is one
    :rtype: Optional[DiscoServer]
    """
    if not ptr:
        return None

    current_server = ptr.contents
    authentication_url_template = current_server.authentication_url_template.decode(
        "utf-8"
    )
    base_url = current_server.base_url.decode("utf-8")
    country_code = current_server.country_code.decode("utf-8")
    display_name = current_server.display_name.decode("utf-8")
    keyword_list = current_server.keyword_list.decode("utf-8")
    public_keys = get_ptr_list_strings(
        lib, current_server.public_key_list, current_server.total_public_keys
    )
    server_type = current_server.server_type.decode("utf-8")
    support_contacts = get_ptr_list_strings(
        lib, current_server.support_contact, current_server.total_support_contact
    )
    return DiscoServer(
        authentication_url_template,
        base_url,
        country_code,
        display_name,
        keyword_list,
        public_keys,
        server_type,
        support_contacts,
    )


def get_disco_servers(lib: CDLL, ptr: c_void_p) -> Optional[DiscoServers]:
    """Gets servers from the Go library in a C structure and returns a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The pointer returned by the Go library for the discovery servers

    :meta private:

    :return: The Discovery Servers if there are any
    :rtype: Optional[DiscoServers]
    """
    if ptr:
        svrs = cast(ptr, POINTER(cDiscoveryServers)).contents

        servers = []

        if svrs.servers:
            for i in range(svrs.total_servers):
                current = get_disco_server(lib, svrs.servers[i])

                if current is None:
                    continue
                servers.append(current)
        disco_version = svrs.version
        lib.FreeDiscoServers(ptr)
        return DiscoServers(disco_version, servers)
    return None


def get_disco_organizations(lib: CDLL, ptr: c_void_p) -> Optional[DiscoOrganizations]:
    """Gets organizations from the Go library in a C structure and returns a Python usable structure

    :param lib: CDLL: The Go shared library
    :param ptr: c_void_p: The pointer returned by the Go library for the discovery organizations

    :meta private:

    :return: The Discovery Organizations if there are any
    :rtype: Optional[DiscoOrganizations]
    """
    if ptr:
        orgs = cast(ptr, POINTER(cDiscoveryOrganizations)).contents
        organizations = []
        if orgs.organizations:
            for i in range(orgs.total_organizations):
                current = get_disco_organization(orgs.organizations[i])
                if current is None:
                    continue
                organizations.append(current)
        disco_version = orgs.version
        lib.FreeDiscoOrganizations(ptr)
        return DiscoOrganizations(disco_version, organizations)
    return None
