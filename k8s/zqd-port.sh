#!/bin/bash
set -x #echo on
# start by killing existing port-forwards
kill $(ps aux | grep '[p]ort-forward' | awk '{print $2}')
sleep 1 # wait a second for the processes to get the signal
kubectl port-forward svc/z-recruiter 8020:9867 &
kubectl port-forward svc/z-root 9867:9867 &
sleep 2 # wait longer for port-forwards to complete
curl http://localhost:8020/recruiter/listfree
zapi ls 
