#!/bin/bash
set -x #echo on
# start by killing existing port-forwards
kill $(ps aux | grep '[p]ort-forward' | awk '{print $2}')
sleep 1 # wait a second for the processes to get the signal
kubectl port-forward svc/zsrv-recruiter 8020:9867 &
kubectl port-forward svc/zsrv-root 9867:9867 &
for i in {1..5} ; do
  if nc -z localhost 8020 </dev/null ; then
    break
  fi
  sleep 1
done
for i in {1..5} ; do
  if nc -z localhost 9867 </dev/null ; then
    break
  fi
  sleep 1
done
curl http://localhost:8020/recruiter/listfree
zapi ls 
