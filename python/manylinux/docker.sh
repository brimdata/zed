#!/usr/bin/env bash

set -ex

repotop="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. >/dev/null 2>&1 && pwd)"
docker run --rm -ti \
  -v $repotop:/io quay.io/pypa/manylinux1_x86_64 \
  /io/python/manylinux/build.sh
