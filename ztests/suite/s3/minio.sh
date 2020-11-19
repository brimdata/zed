#!/bin/bash

mkdir -p data/bucket
portdir=$(mktemp -d)
minio server --writeportfile="$portdir/port" --quiet --address "localhost:0" ./data &
trap "rm -rf $portdir; kill -9 $!" EXIT

# Wait for port file to show up. Minio will write this file once the listener
# has started.
i=0
until [ -f $portdir/port ]; do
  let i+=1
  if [ $i -gt 5 ]; then
    echo "timed out waiting for minio to start"
    exit 1
  fi
  sleep 1
done

port=$(cat $portdir/port)
export AWS_REGION=does-not-matter
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:$port
