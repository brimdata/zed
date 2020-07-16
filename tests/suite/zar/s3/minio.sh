#!/bin/bash

mkdir -p data/bucket
minio server --writeportfile="./port" --quiet --address ":0" ./data &
trap "kill -9 $!" EXIT

# Wait for port file to show up. Minio will write this file once the listener
# has started.
i=0
until [ -f ./port ]; do
  let i+=1
  if [ $i -gt 5 ]; then
    echo "timed out waiting for minio to start"
    exit 1
  fi
  sleep 1
done

port=$(cat port)
export AWS_REGION=us-east-2
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:$port

