// for size_t
#include <stddef.h>

typedef struct discoveryServer {
  const char* authentication_url_template;
  const char* base_url;
  const char* country_code;
  const char* display_name;
  const char* keyword_list;
  const char** public_key_list;
  size_t total_public_keys;
  const char* server_type;
  const char** support_contact;
  size_t total_support_contact;
} discoveryServer;

typedef struct discoveryServers {
  unsigned long long int version;
  discoveryServer** servers;
  size_t total_servers;
} discoveryServers;

typedef struct discoveryOrganization {
  const char* display_name;
  const char* org_id;
  const char* secure_internet_home;
  const char* keyword_list;
} discoveryOrganization;

typedef struct discoveryOrganizations {
  unsigned long long int version;
  discoveryOrganization** organizations;
  size_t total_organizations;
} discoveryOrganizations;
