#!/bin/bash

function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for file \"$file\" to appear"
      echo "zqd log:"
      cat zqd.log
      exit 1
    fi
    sleep 1
  done
}

portdir=$(mktemp -d)
mkdir -p data

zqd listen -l=localhost:0 \
  -data=data \
  -loglevel=warn \
  -portfile="$portdir/zqd" \
  -redis.enabled \
  -immcache.kind=redis \
  &> zqd.log &

zqdpid=$!
awaitfile $portdir/zqd
trap "rm -rf $portdir ; kill -9 $zqdpid &>/dev/null ;" EXIT

export ZQD_HOST=localhost:$(cat $portdir/zqd)

