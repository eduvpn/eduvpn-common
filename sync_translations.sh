#!/bin/sh

# This should be run when the source code is updated

go generate ./...
files=$(find client/locales -iname "out.gotext.json")
for f in $files
do
    dir=$(dirname $f)
    cp "$f" "$dir"/messages.gotext.json
done
