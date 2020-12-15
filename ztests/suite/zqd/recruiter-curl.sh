#!/bin/bash

zqd listen -personality=recruiter -l=localhost:0 -portfile="$portdir/zqd" -loglevel=warn &> zqd.log &
zqdpid=$!
awaitfile $portdir/zqd

trap "rm -rf $portdir; kill -9 $zqdpid" EXIT

export ZQD_HOST=localhost:$(cat $portdir/zqd)
