from eduvpn_common import lib, cDiscoveryOrganizations, cDiscoveryServers, get_ptr_list_strings
from ctypes import cast, POINTER


class DiscoOrganization:
    def __init__(self, display_name, org_id, secure_internet_home, keyword_list):
        self.display_name = display_name
        self.org_id = org_id
        self.secure_internet_home = secure_internet_home
        self.keyword_list = keyword_list

    def __str__(self):
        return self.display_name


class DiscoOrganizations:
    def __init__(self, version, organizations):
        self.version = version
        self.organizations = organizations


class DiscoServer:
    def __init__(
        self,
        authentication_url_template,
        base_url,
        country_code,
        display_name,
        keyword_list,
        public_keys,
        server_type,
        support_contacts,
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
    def __init__(self, version, servers):
        self.version = version
        self.servers = servers


def get_disco_organization(ptr):
    if not ptr:
        return None

    current_organization = ptr.contents
    display_name = current_organization.display_name.decode("utf-8")
    org_id = current_organization.org_id.decode("utf-8")
    secure_internet_home = current_organization.secure_internet_home.decode("utf-8")
    keyword_list = current_organization.keyword_list.decode("utf-8")
    return DiscoOrganization(display_name, org_id, secure_internet_home, keyword_list)


def get_disco_server(ptr):
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
        current_server.public_key_list, current_server.total_public_keys
    )
    server_type = current_server.server_type.decode("utf-8")
    support_contacts = get_ptr_list_strings(
        current_server.support_contact, current_server.total_support_contact
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


def get_disco_servers(ptr):
    if ptr:
        svrs = cast(ptr, POINTER(cDiscoveryServers)).contents

        servers = []

        if svrs.servers:
            for i in range(svrs.total_servers):
                current = get_disco_server(svrs.servers[i])

                if current is None:
                    continue
                servers.append(current)
        lib.FreeDiscoServers(ptr)
        return DiscoServers(svrs.version, servers)
    return None


def get_disco_organizations(ptr):
    if ptr:
        orgs = cast(ptr, POINTER(cDiscoveryOrganizations)).contents
        organizations = []
        if orgs.organizations:
            for i in range(orgs.total_organizations):
                current = get_disco_organization(orgs.organizations[i])
                if current is None:
                    continue
                organizations.append(current)
        lib.FreeDiscoOrganizations(ptr)
        return DiscoOrganizations(orgs.version, organizations)
    return None
