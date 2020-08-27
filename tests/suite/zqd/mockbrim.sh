#!/bin/bash

# This file simulates a running version of brim desktop. It forks a zqd process
# then sits forever on the main thread.

function awaitdeadzqd {
  i=0
  function zqdalive { kill -0 $ZQD_PID 2> /dev/null; }
  while zqdalive ; do
    let i+=1
    if [ $i -gt 5 ]; then
      echo "timed out waiting for zqd to exit" 
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

mkdir -p zqddata
zqddata=zqddata
tempdir=$(mktemp -d)

mockbrim -zqddata="$zqddata" -portfile="$tempdir/port" -pidfile="$tempdir/pid" &
mockbrimpid=$!

# wait for zqd to start
awaitfile $tempdir/port
awaitfile $tempdir/pid

export ZQD_HOST=localhost:$(cat $tempdir/port)
export ZQD_PID=$(cat $tempdir/pid)
export MOCKBRIM_PID=$mockbrimpid

# ensure that zqd process isn't leaked
trap "kill -9 $ZQD_PID 2>/dev/null" EXIT
rm -rf $tempdir
