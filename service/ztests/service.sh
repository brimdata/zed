#!/bin/bash

function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
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

portdir=$(mktemp -d)

zed lake serve -l=localhost:0 -R $lakeroot -portfile=$portdir/lake -log.level=warn $LAKE_EXTRA_FLAGS &> lake.log &
lakepid=$!
awaitfile $portdir/lake

trap "rm -rf $portdir; kill $lakepid;" EXIT

export ZED_LAKE_HOST=localhost:$(cat $portdir/lake)
