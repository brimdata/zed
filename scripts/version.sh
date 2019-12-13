#!/bin/sh
set -e

tag=$(git tag --points-at HEAD --sort=-version:refname --no-column | head -n 1)
if [ -z "$tag" ]; then
    tag=$(git log -1 --pretty=format:%h)
fi

echo "$tag"

