#!/bin/bash
set -x #echo on

kind create cluster --name zq-local
kubectl create namespace zq
kubectl config set-context zq \
  --namespace=zq \
  --cluster=kind-zq-local \
  --user=kind-zq-local
kubectl config use-context zq

