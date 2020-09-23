#!/bin/bash
set -x #echo on

# start by killing existing port-forwards and dashboard
kill $(ps aux | grep '[p]ort-forward' | awk '{print $2}')
kill $(ps aux | grep '[l]inkerd dashboard' | awk '{print $2}')

sleep 1 # wait a second for the processes to get the signal

kubectl port-forward svc/zqd-test-1 9867:9867 &
linkerd dashboard &
