#!/usr/bin/env bash

set -o pipefail -ex

# Prepare Go
curl -sS https://storage.googleapis.com/golang/go1.13.12.linux-amd64.tar.gz | tar xvzf - -C /
export PATH=$PATH:/go/bin

# Build go zqext
make -C /io build-python-lib

# Create & audit wheels for each Python version
for py in cp36-cp36m cp37-cp37m cp38-cp38; do
    "/opt/python/$py/bin/pip" wheel --no-deps --wheel-dir /tmp /io/python
done
ls /tmp/*.whl | xargs -n1 --verbose auditwheel repair --wheel-dir /io/dist
