#!/bin/bash

export ZQD_HOST=localhost:8020

echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"addr":"a.b.c:5000","node":"a.b"}' \
http://$ZQD_HOST/recruiter/register 2> err
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"N":1}' \
http://$ZQD_HOST/recruiter/recruit 2> err
#
# For this second register, the worker will be in the reserved pool,
# so it will not be reregistered.
#
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"addr":"a.b.c:5000","node":"a.b"}' \
http://$ZQD_HOST/recruiter/register 2> err
#
# unreserve the worker
#
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"addr":"a.b.c:5000"}' \
http://$ZQD_HOST/recruiter/unreserve 2> err
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"addr":"a.b.c:5000","node":"a.b"}' \
http://$ZQD_HOST/recruiter/register 2> err
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"addr":"a.b.c:5000"}' \
http://$ZQD_HOST/recruiter/deregister 2> err
echo ===
curl --header "Content-Type: application/json" -request POST \
--data '{"N":1}' \
http://$ZQD_HOST/recruiter/recruit 2> err
