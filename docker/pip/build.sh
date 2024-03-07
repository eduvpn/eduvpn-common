#!/bin/sh

git pull origin --tags
VERSION=$(git tag | sort -V | tail -1)
mkdir -p out/"$VERSION"

# amd64
sudo docker build --build-arg COMMONVERSION="$VERSION" -f amd64.dockerfile -t common-pip-amd64 .
sudo docker run -v "$PWD"/out/"$VERSION":/io --rm -ti common-pip-amd64 bash -c "cp /wheelhouse/* /io"

# arm64
sudo docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
sudo docker build --build-arg COMMONVERSION="$VERSION" -f arm64.dockerfile -t common-pip-arm64 .
sudo docker run -v "$PWD"/out/"$VERSION":/io --rm -ti common-pip-arm64 bash -c "cp /wheelhouse/* /io"
