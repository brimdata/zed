#!/bin/sh
set -x #echo on
set -o errexit

# create registry container unless it already exists
reg_name=kind-registry
reg_port=5000
running=$(docker inspect -f '{{.State.Running}}' "${reg_name}" 2>/dev/null || true)
if [ "${running}" != 'true' ]; then
  docker run \
    -d --restart=always -p "${reg_port}:5000" --name "${reg_name}" \
    registry:2
fi

# create a cluster with the local registry enabled in containerd
kind create cluster  --name zq-local --config=- <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${reg_port}"]
    endpoint = ["http://${reg_name}:${reg_port}"]
EOF

# connect the registry to the cluster network
docker network connect "kind" "${reg_name}" || true
