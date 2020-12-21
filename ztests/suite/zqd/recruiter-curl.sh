#!/bin/bash

zqd listen -personality=recruiter -l=localhost:0 -portfile="$portdir/zqd" -loglevel=warn &> zqd.log &
zqdpid=$!
awaitfile $portdir/zqd

export ZQD_HOST=localhost:$(cat $portdir/zqd)
trap "rm -rf $portdir; kill -9 $zqdpid" EXIT
