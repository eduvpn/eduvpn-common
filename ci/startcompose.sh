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

pushd "$SCRIPT_DIR"/.. || exit 1

# Create self-signed certificate
mkdir -p ci/docker/selfsigned
./ci/docker/createcert.sh


# Up the containers and abort on exit. Also rebuild the necessary steps if there are changes
# You can pass EDUVPN_PODCOMP=1 to use podman-compose instead of docker-compose
compose_cmd="docker-compose"
if [ "$EDUVPN_PODCOMP" ]; then
    compose_cmd="podman-compose"
fi

"$compose_cmd" up --build --force-recreate --abort-on-container-exit

popd || exit 1
