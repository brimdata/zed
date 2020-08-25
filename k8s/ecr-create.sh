#!/bin/bash
set -x #echo on

aws ecr create-repository \
    --image-scanning-configuration scanOnPush=true \
    --repository-name zqd
    