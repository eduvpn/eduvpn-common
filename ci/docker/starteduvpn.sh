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

# Start the preliminary services
mkdir /run/php-fpm
php-fpm --nodaemonize &
crond &
httpd -DFOREGROUND &

# Start the daemon in the background and get the PID
vpn-daemon &
pid_daemon=$!

# Wait a bit
sleep 5

# Snippet from vpn-maint-apply-changes
# Enable & Start WireGuard
rm -rf /etc/wireguard/*
if ! /usr/libexec/vpn-server-node/server-config; then
    exit 1
fi
for F in /etc/wireguard/*.conf
do
    case ${F} in
        *.conf)
            CONFIG_NAME=$(basename "${F}" .conf)
            wg-quick up "${CONFIG_NAME}"
        ;;
    esac
done
# sync with vpn-daemon, no need to wait for the cron, but *ONLY* do this when
# this is a machine with vpn-user-portal installed
if [ -d /etc/vpn-user-portal ]; then
    if [ -f /etc/redhat-release ]; then
        sudo -u apache /usr/libexec/vpn-user-portal/daemon-sync
    fi
    if [ -f /etc/debian_version ]; then
        sudo -u www-data /usr/libexec/vpn-user-portal/daemon-sync
    fi
fi


# Add the user with the env variables
sudo -u apache vpn-user-portal-account --add "${PORTAL_USER}" --password "${PORTAL_PASS}"

# Wait for the daemon to finish
wait $pid_daemon
