#!/bin/bash

export AWS_REGION=us-east-2
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:9000

mkdir -p data/bucket
minio server --quiet --address "localhost:9000" ./data > /dev/null &
trap "kill -9 $!" EXIT
i=0
until nc -vzw 2 localhost 9000; do
  let i+=1
  if [ $i -gt 5 ]; then
    echo "timed out waiting for minio to start"
    exit 1
  fi
  sleep 2
done
