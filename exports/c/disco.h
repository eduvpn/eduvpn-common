// for size_t
#include <stddef.h>

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
