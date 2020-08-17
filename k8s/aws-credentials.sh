#!/bin/bash
set -x #echo on
kubectl delete secret aws-credentials # delete the old ones
kubectl create secret generic aws-credentials \
  --from-literal="aws-region=$(aws configure get region)" \
  --from-literal="aws-access-key-id=$(aws configure get aws_access_key_id)" \
  --from-literal="aws-secret-access-key=$(aws configure get aws_secret_access_key)" \
  --from-literal="aws-session-token=$(aws configure get aws_session_token)"
