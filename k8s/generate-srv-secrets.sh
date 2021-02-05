#!/bin/bash

function genpassword() {
  openssl rand -base64 17 | sed "s/[^a-zA-Z0-9]//g" 
}

kubectl create secret generic postgres \
  --from-literal=postgresql-postgres-password=$(genpassword) \
  --from-literal=postgresql-password=$(genpassword)

kubectl create secret generic redis \
  --from-literal=redis-password=$(genpassword)
