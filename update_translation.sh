#!/usr/bin/env bash

set -e

git diff --quiet client/locales client/zgotext.go || { echo "There are local uncommited locale changes, exiting..."; exit 1; }

tempdir=$(mktemp -d --tmpdir=.)
trap 'rm -rf -- "$tempdir"' EXIT

pushd "$tempdir" >/dev/null

echo "Getting translations..."
wget -q -O translation.zip "https://hosted.weblate.org/download/eduvpn-common/eduvpn-common/?format=zip"
unzip -q translation.zip

echo "Copying translations..."
cp -r eduvpn-common/eduvpn-common/client ../

popd >/dev/null

echo "Checking for changes..."
git diff --quiet client/locales && { echo "No locale changes, exiting..."; exit 1; }

echo "Regenerating locales with go generate ./..."
go generate ./...
git add client/zgotext.go client/locales
git commit -m "Client: Update translations from Weblate"
