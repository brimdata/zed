#!/bin/bash

function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for file \"$file\" to appear"
      for f in *.log ; do
        echo logfile: $f
        cat $f
      done
      exit 1
    fi
    sleep 1
  done
}

zqdroot=zqdroot
mkdir -p $zqdroot

mkdir -p s3/bucket
portdir=$(mktemp -d)
