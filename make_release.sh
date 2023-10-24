#!/bin/sh

# This script was adapted from fkooman: https://git.sr.ht/~fkooman/vpn-daemon/tree/main/item/make_release.sh. Thanks!
#
# Make a release of the version specified in internal/version/version.go and automatically release the artifacts
#

# Fail if error
set -e

# change these to your liking
BRANCH=v1
IS_PRERELEASE=true
REPO="eduvpn/eduvpn-common"

# make a fresh clone
rm -rf /tmp/eduvpn-common
git clone "https://github.com/$REPO" /tmp/eduvpn-common
cd /tmp/eduvpn-common
git checkout ${BRANCH}

echo "building $(git log -n 1 | head -n 1)"
PROJECT_NAME=$(basename "${PWD}")
PROJECT_VERSION=$(grep -o 'const Version = "[^"]*' internal/version/version.go | cut -d '"' -f 2)
RELEASE_DIR="${PWD}/release"
KEY_ID=227FF3F8F829D9A9314D9EBA02BB8048BBFF222C
if [ "$IS_PRERELEASE" = true ]; then
    KEY_ID=7A73D62AD0F084571A32C960D57104BF9B223CBF
fi

if ! command -v "tar" &>/dev/null; then
    echo "please install tar for archiving the code"
    exit 1
fi

if ! command -v "gh" &>/dev/null; then
    echo "please install the GitHub CLI for creating releases"
    exit 1
fi

if ! command -v "wget" &>/dev/null; then
    echo "please install wget for getting the discovery files"
    exit 1
fi

if ! command -v "gpg" &>/dev/null; then
    echo "please install gpg for signing the archive"
    exit 1
fi

if ! command -v "minisign" &>/dev/null; then
    echo "please install minisign for signing the archive"
    exit 1
fi

if [ "$(git tag -l "${PROJECT_VERSION}")" ]; then
    echo "Version: ${PROJECT_VERSION} already has a tag"
    exit 1
fi

mkdir -p "${RELEASE_DIR}"

if [ -f "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz" ]; then
    echo "Version ${PROJECT_VERSION} already has a release!"
    exit 1
fi

# Archive repository
git archive --prefix "${PROJECT_NAME}-${PROJECT_VERSION}/" "${BRANCH}" | tar -xf -

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

# upload the release with the github cli tool, this also creates a tag
CHANGES_TRIM=$(sed "/^# $PROJECT_VERSION/,/^#/!d;//d" "CHANGES.md")

if [ "$IS_PRERELEASE" = true ]; then
    CHANGES=$(printf "These pre-releases are signed with \`keys/app+linux+dev@eduvpn.org.asc\` and \`keys/minisign-CA9409316AC93C07.pub\`\nChangelog:\n%s" "${CHANGES_TRIM}")
    gh release create "${PROJECT_VERSION}" --target "${BRANCH}" --title "${PROJECT_VERSION}" --notes "${CHANGES}" --prerelease --repo "$REPO"
else
    CHANGES=$(printf "These releases are signed with \`keys/app+linux@eduvpn.org.asc\` and \`keys/minisign-CA9409316AC93C07.pub\`\nChangelog:\n%s" "${CHANGES_TRIM}")
    gh release create "${PROJECT_VERSION}" --target "${BRANCH}" --title "${PROJECT_VERSION}" --notes "${CHANGES}" --repo "$REPO"
fi
gh release upload "${PROJECT_VERSION}" "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz" --repo "$REPO" "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz.asc" "${RELEASE_DIR}/${PROJECT_NAME}-${PROJECT_VERSION}.tar.xz.minisig"
