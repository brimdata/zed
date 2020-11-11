#!/bin/bash
set -x #echo on

helm uninstall zqd
make install
make docker-push-ecr
sleep 3
make helm-install-with-ingress

kill $(ps aux | grep '[p]ort-forward' | awk '{print $2}')
kubectl port-forward svc/zqd 9867:9867 &

zapi ls
