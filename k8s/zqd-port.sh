#!/bin/bash
set -x #echo on
# start by killing existing port-forwards
kill $(ps aux | grep '[p]ort-forward' | awk '{print $2}')
sleep 1 # wait a second for the processes to get the signal
source zsrv-port-forward.sh
curl http://localhost:8020/recruiter/listfree
zapi ls 
