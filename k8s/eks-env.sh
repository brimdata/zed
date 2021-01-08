#!/bin/bash

account=$(aws sts get-caller-identity --output text --query Account)
arn=$(aws sts get-caller-identity --output text --query Arn)
user=${arn#*/}

export ZQD_DATA_URI=s3://zqd-demo-1/${user}/zqd-meta
export ZQD_ECR_HOST=${account}.dkr.ecr.us-east-2.amazonaws.com
export ZQD_K8S_USER=$user
export ZQD_TEST_CLUSTER=zq-test.us-east-2.eksctl.io
