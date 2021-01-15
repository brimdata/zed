#!/bin/bash
# This is designed to be run from the zq root directory

make install
helm uninstall z
make docker-push-ecr

sleep 5 # wait for ECR to catch up a little

make helm-install

sleep 15 # usually is enough on my laptop

./k8s/zqd-port.sh

# setup in advance:
zapi new -k archivestore -d s3://brim-scratch/mark/sp-m1 -thresh 5MB sp-m1
zapi -s sp-m1 post s3://brim-scratch/mark/conn.log.gz

zapi -s sp-m1 get -workers 2 -t "count()"
zapi -s sp-m1 get -workers 2 -t "39161"

# To add more workers:
# kubectl scale --replicas=3 deployment/worker-zqd
