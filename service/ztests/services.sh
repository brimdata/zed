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
      echo "lake serve log:"
      cat lake.log
      exit 1
    fi
    sleep 1
  done
}

lakeroot=$1
if [ -z "$lakeroot" ]; then
  lakeroot=lakeroot
fi
mkdir -p $lakeroot

mkdir -p s3/bucket
portdir=$(mktemp -d)

minio server --writeportfile=$portdir/minio --quiet --address localhost:0 ./s3 > minio.log 2>&1 &
miniopid=$!
awaitfile $portdir/minio

# AWS env variables must be set before zed lake serve starts so zed lake has access
# to them.
export AWS_REGION=does-not-matter
export AWS_ACCESS_KEY_ID=minioadmin
export AWS_SECRET_ACCESS_KEY=minioadmin
export AWS_S3_ENDPOINT=http://localhost:$(cat $portdir/minio)

zed lake serve -l=localhost:0 -R $lakeroot -portfile=$portdir/lake -log.level=info $LAKE_EXTRA_FLAGS &> lake.log &
lakepid=$!
awaitfile $portdir/lake

trap "rm -rf $portdir; kill -9 $miniopid $lakepid;" EXIT

export LAKE_HOST=localhost:$(cat $portdir/lake)
