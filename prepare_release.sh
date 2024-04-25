#!/bin/sh

set -e

PROJECT_VERSION="$1"

if [ -z "$1" ]; then
    echo "No version supplied"
    exit 1
fi

if ! command -v "ruff" &>/dev/null; then
    echo "please install ruff for formatting the python wrapper"
    exit 1
fi

if ! command -v "gofumpt" &>/dev/null; then
    echo "please install gofumpt for formatting the go code"
    exit 1
fi

if [ "$(git tag -l "${PROJECT_VERSION}")" ]; then
    echo "Version: ${PROJECT_VERSION} already has a tag"
    exit 1
fi

# First check if th
if [[ $(git diff) ]]; then
   echo "There are changes, commit them before releasing"
   exit 1
fi

# Format all go files
git ls-files | grep '.go$' | xargs -I {} gofumpt -w {} >/dev/null
if [[ $(git diff) ]]; then
    git add -u
    git commit -m "Format: Run Gofumpt"
fi

# Format all Python files
git ls-files | grep '.py$' | xargs -I {} ruff format --quiet {} >/dev/null
if [[ $(git diff) ]]; then
    git add -u
    git commit -m "Format: Run Ruff"
fi

# Replace version number
# replace in internal/version
sed -i "s/const Version = ".*"/const Version = \"${PROJECT_VERSION}\"/" internal/version/version.go
sed -i "s/__version__ = ".*"/__version__ = \"${PROJECT_VERSION}\"/" wrappers/python/setup.py
sed -i "s/__version__ = ".*"/__version__ = \"${PROJECT_VERSION}\"/" wrappers/python/eduvpn_common/__init__.py

git add -u
git commit -m "Version: Update to ${PROJECT_VERSION}"
