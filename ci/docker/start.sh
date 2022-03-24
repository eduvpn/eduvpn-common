#!/usr/bin/env bash

systemctl start php-fpm
systemctl start httpd
systemctl start crond

vpn-daemon &
sleep 5

vpn-maint-apply-changes

USER_NAME="docker"
USER_PASS="docker"

sudo -u apache vpn-user-portal-account --add "${USER_NAME}" --password "${USER_PASS}"
