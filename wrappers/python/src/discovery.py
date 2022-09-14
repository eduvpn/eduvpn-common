from . import lib, cDiscoveryOrganizations
from ctypes import cast, POINTER


class DiscoOrganization:
    def __init__(self, display_name, org_id, secure_internet_home, keyword_list):
        self.display_name = display_name
        self.org_id = org_id
        self.secure_internet_home = secure_internet_home
        self.keyword_list = keyword_list


class DiscoOrganizations:
    def __init__(self, version, organizations):
        self.version = version
        self.organizations = organizations


def get_disco_organization(ptr):
    if not ptr:
        return None

    current_organization = ptr.contents
    display_name = current_organization.display_name.decode("utf-8")
    org_id = current_organization.org_id.decode("utf-8")
    secure_internet_home = current_organization.secure_internet_home.decode("utf-8")
    keyword_list = current_organization.keyword_list.decode("utf-8")
    return DiscoOrganization(display_name, org_id, secure_internet_home, keyword_list)


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
