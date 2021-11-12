#!/bin/bash

# This file simulates a running version of brim desktop. It forks a service process
# then sits forever on the main thread.

function awaitdeadservice {
  i=0
  function servicealive { kill -0 $LAKE_PID 2> /dev/null; }
  while servicealive ; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for service to exit" 
      exit 1
    fi
    sleep 1
  done
}

function awaitfile {
  file=$1
  i=0
  until [ -f $file ]; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for file \"$file\" to appear"
      exit 1
    fi
    sleep 1
  done
}

mkdir -p lakeroot
lakeroot=lakeroot
tempdir=$(mktemp -d)

mockbrim -R="$lakeroot" -portfile="$tempdir/port" -pidfile="$tempdir/pid" &
mockbrimpid=$!

# wait for service to start
awaitfile $tempdir/port
awaitfile $tempdir/pid

export ZED_LAKE=http://localhost:$(cat $tempdir/port)
export LAKE_PID=$(cat $tempdir/pid)
export MOCKBRIM_PID=$mockbrimpid

# ensure that lake service process isn't leaked
trap "kill -9 $LAKE_PID 2>/dev/null || :" EXIT
rm -rf $tempdir
