#!/bin/bash
set -x #echo on

# This shell file helps set up ad-hoc testing of a local
# "cluster" of zqd processes. It is helpful for trouble-shooting.
# it should be run in a seperate directory that contains smtp.log.gz

# remove processes from previous run of dist-test.sh:
kill $(ps aux | grep '[z]qd' | awk '{print $2}')
kill $(ps aux | grep '[t]ail' | awk '{print $2}')

# remove directories from previous run of dist-test.sh:
rm -rf spacedir

mkdir -p spacedir
# Start 2 zqd/workers
zqd listen -l :9871 -data spacedir &> zqd-w1.log &
zqd listen -l :9872 -data spacedir &> zqd-w2.log &
# start the root zqd
zqd listen -l :9867 -data spacedir -workers :9871,:9872  &> zqd-root.log &
# tail everything
tail -f zqd-w1.log &
tail -f zqd-w2.log &
tail -f zqd-root.log &

sleep 1

# create an archive with the records from smtp.log.gz
zapi new -k archivestore -d spacedir -thresh 15KB testsp
zapi -s testsp post smtp.log.gz

sleep 1
echo "Here is a example zapi query that uses the cluster:"
zapi -s testsp get -t "count()"
