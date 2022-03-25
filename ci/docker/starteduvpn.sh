#!/usr/bin/env bash

if [[ -z "${PORTAL_USER}" ]]; then
    printf "Error: No portal username set, set the PORTAL_USER env var\n"
    exit 1
fi

if [[ -z "${PORTAL_PASS}" ]]; then
    printf "Error: No portal username set, set the PORTAL_PASS env var\n"
    exit 1
fi

systemctl start php-fpm
systemctl start httpd
systemctl start crond

vpn-daemon &
pid_daemon=$!
sleep 5

vpn-maint-apply-changes

sudo -u apache vpn-user-portal-account --add "${PORTAL_USER}" --password "${PORTAL_PASS}"

wait $pid_daemon
