#!/bin/bash
set -x #echo on

for i in $(seq 1 100)
do
  : "Iteration $i:"
  zapi -s http-space-2 get "tail 1"
  sleep 60
done
