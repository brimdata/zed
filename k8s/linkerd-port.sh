#!/bin/bash
set -x #echo on
# start by killing existing dashboard
kill $(ps aux | grep '[l]inkerd dashboard' | awk '{print $2}')
sleep 1 # wait a second for the processes to get the signal
linkerd dashboard &
