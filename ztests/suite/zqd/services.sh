#!/bin/bash

function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for file \"$file\" to appear"
      echo "minio log:"
      cat minio.log
      echo "zqd log:"
      cat zqd.log
      exit 1
    fi
    sleep 1
  done
}

zqdroot=$1
if [ -z "$zqdroot" ]; then
  zqdroot=zqdroot
fi
mkdir -p $zqdroot

mkdir -p s3/bucket
portdir=$(mktemp -d)

minio server --writeportfile=$portdir/minio --quiet --address localhost:0 ./s3 > minio.log 2>&1 &
miniopid=$!
awaitfile $portdir/minio

# AWS env variables must be set before zqd starts so zqd has access to them.
export AWS_REGION=does-not-matter
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:$(cat $portdir/minio)

zqd listen -l=localhost:0 -portfile=$portdir/zqd -data=$zqdroot -loglevel=warn -suricataupdater=true &> zqd.log &
zqdpid=$!
awaitfile $portdir/zqd

trap "rm -rf $portdir; kill -9 $miniopid $zqdpid" EXIT

export ZQD_HOST=localhost:$(cat $portdir/zqd)
