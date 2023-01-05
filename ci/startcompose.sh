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

pushd "$SCRIPT_DIR"/..

# Create self-signed certificate
mkdir -p ci/docker/selfsigned
./ci/docker/createcert.sh

# Up the containers and abort on exit. Also rebuild the necessary steps if there are changes
# You can symlink docker-compose to podman-compose to use Podman
docker-compose up --build --force-recreate --abort-on-container-exit

popd
