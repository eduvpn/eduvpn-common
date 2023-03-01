#!/bin/sh

# This script was adapted from fkooman: https://git.sr.ht/~fkooman/vpn-daemon/tree/main/item/make_release.sh. Thanks!
#
# Make a release of the latest tag, or of the tag/branch/commit specified as
# the first parameter.
#

# Fail if error
set -e

PROJECT_NAME=$(basename "${PWD}")
PROJECT_VERSION=${1}
RELEASE_DIR="${PWD}/release"
KEY_ID=227FF3F8F829D9A9314D9EBA02BB8048BBFF222C

mkdir -p "${RELEASE_DIR}"

if [ -z "${1}" ]; then
    # we take the last "tag" of the Git repository as version
    PROJECT_VERSION=$(git describe --abbrev=0 --tags)
    echo Version: "${PROJECT_VERSION}"
fi

if [ -f "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz" ]; then
    echo "Version ${PROJECT_VERSION} already has a release!"

    exit 1
fi

# Archive repository
git archive --prefix "${PROJECT_NAME}-${PROJECT_VERSION}/" "${PROJECT_VERSION}" | tar -xf -

# We run "make vendor" in it to add all dependencies to the vendor directory
# so we have a self contained source archive.
cd "${PROJECT_NAME}-${PROJECT_VERSION}"
go mod vendor

# Get discovery files and verify signature
echo "getting and verifying discovery files..."
wget -q https://disco.eduvpn.org/v2/organization_list.json -O internal/discovery/organization_list.json
wget -q https://disco.eduvpn.org/v2/organization_list.json.minisig -O internal/discovery/organization_list.json.minisig
minisign -Vm "internal/discovery/organization_list.json" -P RWRtBSX1alxyGX+Xn3LuZnWUT0w//B6EmTJvgaAxBMYzlQeI+jdrO6KF || minisign -Vm "internal/discovery/organization_list.json" -P RWQKqtqvd0R7rUDp0rWzbtYPA3towPWcLDCl7eY9pBMMI/ohCmrS0WiM
wget -q https://disco.eduvpn.org/v2/server_list.json -O internal/discovery/server_list.json
wget -q https://disco.eduvpn.org/v2/server_list.json.minisig -O internal/discovery/server_list.json.minisig
minisign -Vm "internal/discovery/server_list.json" -P RWRtBSX1alxyGX+Xn3LuZnWUT0w//B6EmTJvgaAxBMYzlQeI+jdrO6KF || minisign -Vm "internal/discovery/server_list.json" -P RWQKqtqvd0R7rUDp0rWzbtYPA3towPWcLDCl7eY9pBMMI/ohCmrS0WiM
cd ..
tar -cJf "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz" "${PROJECT_NAME}-${PROJECT_VERSION}"
rm -rf "${PROJECT_NAME}-${PROJECT_VERSION}"

echo "signing using gpg and minisign"
gpg --default-key ${KEY_ID} --armor --detach-sign "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz"
minisign -Sm "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz"
