#!/bin/bash
set -x #echo on
set -o errexit

kubectl create namespace zq
kubectl config set-context zq \
  --namespace=zq \
  --cluster=kind-zq-local \
  --user=kind-zq-local
kubectl config use-context zq

