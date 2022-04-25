#!/bin/sh

WEB_FQDN="eduvpnserver"

# Create self signed cert and key
openssl req \
	-nodes \
	-subj "/CN=${WEB_FQDN}" \
	-x509 \
	-sha256 \
	-newkey rsa:2048 \
	-keyout "./selfsigned/${WEB_FQDN}.key" \
	-out "./selfsigned/${WEB_FQDN}.crt" \
	-days 90
