#ifndef SERVER_H
#define SERVER_H

// The struct for a single server profile
typedef struct serverProfile {
  const char* id;
  const char* display_name;
  int default_gateway;
  const char** dns_search_domains;
  size_t total_dns_search_domains;
} serverProfile;

// The struct for all server profiles
typedef struct serverProfiles {
  int current;
  serverProfile** profiles;
  size_t total_profiles;
} serverProfiles;

// The struct for server locations
typedef struct serverLocations {
  const char** locations;
  size_t total_locations;
} serverLocations;

// The struct for a single server
typedef struct server {
  const char* identifier;
  const char* display_name;
  const char* server_type;
  const char* country_code;
  const char** support_contact;
  size_t total_support_contact;
  serverLocations* locations;
  serverProfiles* profiles;
  unsigned long long int expire_time;
} server;

// The struct for all servers
typedef struct servers {
  server** custom_servers;
  size_t total_custom;
  server** institute_servers;
  size_t total_institute;
  server* secure_internet_server;
} servers;

#endif /* GRANDPARENT_H */
