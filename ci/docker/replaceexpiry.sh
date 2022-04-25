#!/usr/bin/env bash

# If no custom expiry set, do nothing
[ -z "${OAUTH_EXPIRED_TTL}" ] && exit

# Replace oauth expiry
sed -i "s/return \[/return \[\n'Api' => [\n'tokenExpiry' => 'PT${OAUTH_EXPIRED_TTL}S',\n],/g" "$1"
