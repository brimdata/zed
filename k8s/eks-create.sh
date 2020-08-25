#!/bin/bash
set -x #echo on

eksctl create cluster \
--name zqtest \
--version 1.17 \
--nodegroup-name standard-workers \
--node-type t3.medium \
--nodes 1 \
--nodes-min 1 \
--nodes-max 3 \
--ssh-access \
--ssh-public-key zq-eks-test.pub \
--managed \
--asg-access
