#!/bin/bash

# (1) Start a zqd recruiter process.
zqd listen -personality=recruiter -l=localhost:0 -portfile=$portdir/zqd-rec &> zqd-rec.log &
zqdrpid=$!
awaitfile $portdir/zqd-rec

recruiter=localhost:$(cat $portdir/zqd-rec)
node_name=test1

# (2) Start two zqd workers that register with the recruiter.
zqd listen -personality=worker -l=localhost:0 -portfile=$portdir/zqd-w1 \
  -worker.recruiter=$recruiter -worker.node=$node_name &> zqd-w1.log &
zqdw1pid=$!
awaitfile $portdir/zqd-w1

zqd listen -personality=worker -l=localhost:0 -portfile=$portdir/zqd-w2 \
  -worker.recruiter=$recruiter -worker.node=$node_name &> zqd-w2.log &
zqdw2pid=$!
awaitfile $portdir/zqd-w2

# (3) Start a zqd "root" process (default personality for now) that may recruit workers.

zqd listen -l=localhost:0 -portfile=$portdir/zqd-root -data=$zqdroot \
  -worker.recruiter=$recruiter -worker.fallback=false &> zqd-root.log &
zqdrootpid=$!
awaitfile $portdir/zqd-root

export ZQD_HOST=localhost:$(cat $portdir/zqd-root)
trap "rm -rf $portdir; kill -9 $zqdrpid $zqdw1pid $zqdw2pid $zqdrootpid" EXIT
