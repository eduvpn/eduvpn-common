#ifndef EXPORTS_H
#define EXPORTS_H

#include <stdint.h>
#include <stdlib.h>

typedef long long int (*ReadRxBytes)();

typedef int (*StateCB)(int oldstate, int newstate, void* data);

typedef void (*RefreshList)();
typedef void (*TokenGetter)(const char* server_id, int server_type, char* out, size_t len);
typedef void (*TokenSetter)(const char* server_id, int server_type, const char* tokens);
typedef void (*ProxySetup)(int fd);

static long long int get_read_rx_bytes(ReadRxBytes read)
{
    return read();
}
static int call_callback(StateCB callback, int oldstate, int newstate, void* data)
{
    return callback(oldstate, newstate, data);
}
static void call_refresh_list(RefreshList refresh)
{
    refresh();
}
static void call_token_getter(TokenGetter getter, const char* server_id, int server_type, char* out, size_t len)
{
    getter(server_id, server_type, out, len);
}
static void call_token_setter(TokenSetter setter, const char* server_id, int server_type, const char* tokens)
{
    setter(server_id, server_type, tokens);
}
static void call_proxy_setup(ProxySetup proxysetup, int fd)
{
    proxysetup(fd);
}

#endif /* EXPORTS_H */
