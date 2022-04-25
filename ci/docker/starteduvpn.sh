#!/usr/bin/env bash

# Check if credentials are set
# If not fail with exit code 1
if [[ -z "${PORTAL_USER}" ]]; then
    printf "Error: No portal username set, set the PORTAL_USER env var\n"
    exit 1
fi

if [[ -z "${PORTAL_PASS}" ]]; then
    printf "Error: No portal username set, set the PORTAL_PASS env var\n"
    exit 1
fi

# Replace expiry
./replaceexpiry.sh /etc/vpn-user-portal/config.php

# Start the preliminary systemd units
systemctl start php-fpm
systemctl start httpd
systemctl start crond

# Start the daemon in the background and get the PID
vpn-daemon &
pid_daemon=$!

# Wait a bit
sleep 5

# Apply the vpn configuration
vpn-maint-apply-changes

# Add the user with the env variables
sudo -u apache vpn-user-portal-account --add "${PORTAL_USER}" --password "${PORTAL_PASS}"

# Wait for the daemon to finish
wait $pid_daemon
