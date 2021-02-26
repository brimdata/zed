#!/bin/bash
set -x #echo on

# Derived from:
#https://dev.to/hayderimran7/create-a-simple-vpc-peer-between-kubernetes-and-rds-postgres-lhn

DRY_RUN=$3

EKS_CLUSTER=$1
EKS_VPC="$EKS_CLUSTER"/VPC
EKS_PUBLIC_ROUTING_TABLE="$EKS_CLUSTER"/PublicRouteTable

RDS_NAME=$2
RDS_VPC="$RDS_NAME"/VPC
RDS_MAIN_ROUTE_TABLE="$RDS_NAME"/MainRouteTable
RDS_DB_NAME="aurora-zq-test-instance-1"

PEERING_NAME="$RDS_NAME-peer-$EKS_CLUSTER"

echo "getting VPC ID and CIDR of acceptor (RDS instance)"
ACCEPT_VPC_ID=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=$RDS_VPC \
  --query=Vpcs[0].VpcId --output text)
ACCEPT_CIDR=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=$RDS_VPC \
  --query=Vpcs[0].CidrBlockAssociationSet[0].CidrBlock --output text)

echo "getting VPC ID and CIDR of requestor (EKS)"
REQUEST_VPC_ID=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=$EKS_VPC \
    --query=Vpcs[0].VpcId --output text)
REQUEST_CIDR=$(aws ec2 describe-vpcs --filters Name=tag:Name,Values=$EKS_VPC \
  --query=Vpcs[0].CidrBlockAssociationSet[0].CidrBlock --output text)

# get Public Route table ID of requestor and acceptor
REQ_ROUTE_ID=$(aws ec2 describe-route-tables \
--filters Name=tag:Name,Values=$EKS_PUBLIC_ROUTING_TABLE \
--query=RouteTables[0].RouteTableId --output text)
ACCEPT_ROUTE_ID=$(aws ec2 describe-route-tables \
  --filters Name=tag:Name,Values=$RDS_MAIN_ROUTE_TABLE \
  --query=RouteTables[0].RouteTableId --output text)

# Create Peering Connection
peerVPCID=$(aws $DRY_RUN ec2 create-vpc-peering-connection \
  --vpc-id $REQUEST_VPC_ID --peer-vpc-id $ACCEPT_VPC_ID \
  --query VpcPeeringConnection.VpcPeeringConnectionId --output text)
aws $DRY_RUN ec2 accept-vpc-peering-connection --vpc-peering-connection-id "$peerVPCID"
aws $DRY_RUN ec2 create-tags --resources "$peerVPCID" --tags "Key=Name,Value=$PEERING_NAME"

# Adding the private VPC CIDR block to our public VPC route table as destination
aws $DRY_RUN ec2 create-route --route-table-id "$REQ_ROUTE_ID" \
  --destination-cidr-block "$ACCEPT_CIDR" --vpc-peering-connection-id "$peerVPCID"
aws $DRY_RUN ec2 create-route --route-table-id "$ACCEPT_ROUTE_ID" \
  --destination-cidr-block "$REQUEST_CIDR" --vpc-peering-connection-id "$peerVPCID"

# Allow DNS resolution (for RDS host)
aws $DRY_RUN ec2 modify-vpc-peering-connection-options \
  --vpc-peering-connection-id "$peerVPCID" \
  --requester-peering-connection-options '{"AllowDnsResolutionFromRemoteVpc":true}' \
  --accepter-peering-connection-options '{"AllowDnsResolutionFromRemoteVpc":true}' 

# Add a rule that allows inbound RDS (from our Public Instance source)
RDS_VPC_SECURITY_GROUP_ID=$(aws rds describe-db-instances \
  --db-instance-identifier $RDS_DB_NAME \
  --query=DBInstances[0].VpcSecurityGroups[0].VpcSecurityGroupId --output text)
aws $DRY_RUN ec2 authorize-security-group-ingress \
  --group-id ${RDS_VPC_SECURITY_GROUP_ID} \
  --protocol tcp --port 5432 --cidr "$REQUEST_CIDR"
