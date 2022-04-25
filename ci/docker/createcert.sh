#!/usr/bin/env bash
# Get absolute path to current directory this script is in
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

WEB_FQDN="eduvpnserver"

# Create self signed cert and key
openssl req \
	-nodes \
	-subj "/CN=${WEB_FQDN}" \
	-x509 \
	-sha256 \
	-newkey rsa:2048 \
	-keyout "${SCRIPT_DIR}/selfsigned/${WEB_FQDN}.key" \
	-out "${SCRIPT_DIR}/selfsigned/${WEB_FQDN}.crt" \
	-days 90
