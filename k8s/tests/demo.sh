#!/bin/bash
# This is designed to be run from the zq root directory

make install

helm uninstall worker
helm uninstall zqd-root
helm uninstall recruiter

make docker-push-ecr

sleep 5 # wait for ECR to catch up a little

make helm-install-recruiter
make helm-install-root
make helm-install-workers

sleep 10

./k8s/zqd-port.sh

zapi -s sp-m1 get -workers 2 -t "39161"

kubectl scale --replicas=3 deployment/worker-zqd
