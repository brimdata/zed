#!/bin/bash

if ! command -v jq &> /dev/null
then
    echo "jq could not be found"
    exit
fi

if ! command -v yq &> /dev/null
then
    echo "yq could not be found"
    exit
fi

id=$(aws sts get-caller-identity)
userarn=$(echo ${id} | jq '.Arn')
accountid=$(echo ${id} | jq --raw-output '.Account')
username=$(kubectl get -o json -n kube-system configmap aws-auth \
	| jq --raw-output '.data.mapUsers' \
	| yq eval ".[] | select(.userarn == $userarn) | .username" -)

export ZQD_ECR_HOST=${accountid}.dkr.ecr.us-east-2.amazonaws.com
export ZQD_DATA_URI=s3://zqd-demo-1/${username}/zqd-meta
export ZQD_K8S_USER=$username
export ZQD_TEST_CLUSTER=zq-test.us-east-2.eksctl.io
