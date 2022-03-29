#!/usr/bin/env bash

if [[ -z "${PORTAL_USER}" ]]; then
    printf "Error: No portal username set, set the PORTAL_USER env var\n"
    exit 1
fi

if [[ -z "${PORTAL_PASS}" ]]; then
    printf "Error: No portal username set, set the PORTAL_PASS env var\n"
    exit 1
fi

# Get absolute path to current directory this script is in
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Get the parent directory to get the root directory
docker-compose --file ci/docker/docker-compose.yml --project-directory "$SCRIPT_DIR"/.. up --build --force-recreate --abort-on-container-exit
