#!/bin/bash

# start two zqd workers and a zqd root process
zqd listen -l=localhost:0 -portfile=$portdir/zqd-w1 -data=$zqdroot -loglevel=warn -suricataupdater=true &> zqd-w1.log &
zqdw1pid=$!
zqd listen -l=localhost:0 -portfile=$portdir/zqd-w2 -data=$zqdroot -loglevel=warn -suricataupdater=true &> zqd-w2.log &
zqdw2pid=$!

awaitfile $portdir/zqd-w1
portw1=$(cat $portdir/zqd-w1)

awaitfile $portdir/zqd-w2
portw2=$(cat $portdir/zqd-w2)

test_workers=127.0.0.1:$portw1,127.0.0.1:$portw2
zqd listen -l=localhost:0 -portfile=$portdir/zqd -data=$zqdroot \
  -loglevel=warn -suricataupdater=true \
  -worker.bound=$test_workers -worker.fallback=false &> zqd-root.log &
zqdpid=$!
awaitfile $portdir/zqd

export ZQD_HOST=localhost:$(cat $portdir/zqd)
trap "rm -rf $portdir; kill -9 $zqdpid $zqdw1pid $zqdw2pid" EXIT
