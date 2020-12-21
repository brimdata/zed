#!/bin/bash

# first start a zqd recruiter process
zqd listen -personality=recruiter -l=localhost:0 -portfile="$portdir/zqd" -loglevel=warn &> zqd.log &
zqdpid=$!
awaitfile $portdir/zqd

# then start one zqd worker that will register with the recruiter
recruiter="127.0.0.1:$(cat $portdir/zqd)" 
node_name=test1 

zqd listen  -personality=worker -l=localhost:0 -portfile="$portdir/zqd-w1" \
  -data="$zqdroot" -suricataupdater=true  -loglevel=warn \
  -worker.recruiter=$recruiter -worker.node=$node_name -worker.host=$node_name &> zqd-w1.log &
zqdw1pid=$!
awaitfile $portdir/zqd-w1

export ZQD_HOST=localhost:$(cat $portdir/zqd)
trap "rm -rf $portdir; kill -9 $zqdpid $zqdw1pid" EXIT
