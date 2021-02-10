# Using Aurora with Z services (zsrv)

The setup and connection steps below are based on the AWS doc at:

https://aws.amazon.com/getting-started/tutorials/create-connect-postgresql-db/

Modified to work with Aurora PostgresQL. I did all these by hand, because many of the commands have significant side effects and I wanted to monitor them carefully.

## Setting up pod access to an RDS instance

This is based on CLI commands in: https://www.eksworkshop.com/beginner/115_sg-per-pod/

Modified to use our existing EKS cluster.

Start with:
```
aws eks list-clusters
```
Which should display our zq-test cluster.

The next sequence of commands creates and modifies the security groups that we will use for Aurora.

```
export VPC_ID=$(aws eks describe-cluster \
    --name zq-test \
    --query "cluster.resourcesVpcConfig.vpcId" \
    --output text)

echo $VPC_ID

# create RDS security group
aws ec2 create-security-group \
    --description 'RDS SG' \
    --group-name 'RDS_SG' \
    --vpc-id ${VPC_ID}

# save the security group ID for future use
export RDS_SG=$(aws ec2 describe-security-groups \
    --filters Name=group-name,Values=RDS_SG Name=vpc-id,Values=${VPC_ID} \
    --query "SecurityGroups[0].GroupId" --output text)

echo "RDS security group ID: ${RDS_SG}"

# create the POD security group
aws ec2 create-security-group \
    --description 'POD SG' \
    --group-name 'POD_SG' \
    --vpc-id ${VPC_ID}

# save the security group ID for future use
export POD_SG=$(aws ec2 describe-security-groups \
    --filters Name=group-name,Values=POD_SG Name=vpc-id,Values=${VPC_ID} \
    --query "SecurityGroups[0].GroupId" --output text)
echo "POD security group ID: ${POD_SG}"

export POD_SG=$(aws ec2 describe-security-groups \
    --filters Name=group-name,Values=POD_* Name=vpc-id,Values=${VPC_ID} \
    --query "SecurityGroups[0].GroupId" --output text)

export NODE_GROUP_SG=$(aws ec2 describe-security-groups | 
  jq -c '.SecurityGroups[] | select(.GroupName | contains("sg-zq-test")) | .GroupId' -r)
echo "Node Group security group ID: ${NODE_GROUP_SG}"

# allow POD_SG to connect to NODE_GROUP_SG using TCP 53
aws ec2 authorize-security-group-ingress \
    --group-id ${NODE_GROUP_SG} \
    --protocol tcp \
    --port 53 \
    --source-group ${POD_SG}

# allow POD_SG to connect to NODE_GROUP_SG using UDP 53
aws ec2 authorize-security-group-ingress \
    --group-id ${NODE_GROUP_SG} \
    --protocol udp \
    --port 53 \
    --source-group ${POD_SG}

# Allow POD_SG to connect to the RDS
aws ec2 authorize-security-group-ingress \
    --group-id ${RDS_SG} \
    --protocol tcp \
    --port 5432 \
    --source-group ${POD_SG}
```

Now, using the VPC_ID from above, we create a DB subnet group for the Aurora instance.
```
export PUBLIC_SUBNETS_ID=$(aws ec2 describe-subnets \
    --filters "Name=vpc-id,Values=$VPC_ID" "Name=tag:Name,Values=eksctl-zq-test-cluster/SubnetPublic*" \
    --query 'Subnets[*].SubnetId' \
    --output json | jq -c .)
echo "Public Subnets: ${PUBLIC_SUBNETS_ID}"

# create a db subnet group
aws rds create-db-subnet-group \
    --db-subnet-group-name aurora-zq-test \
    --db-subnet-group-description aurora-zq-test \
    --subnet-ids ${PUBLIC_SUBNETS_ID}
```

## Notes on creating AWS Aurora Postgres instance for test cluster

I used the AWS console for these commands, and copied in the values from the environment variable in the previous section.

1. Choose:RDS Create Database
1. Select: Standard Create
1. Select: Amazon Aurora with PostgreSQL compatibility
1. Select: Serverless
1. DB cluster id: aurora-zq-test
1. Username: postgres
1. Password: Autogenerate
1. Capacity Settings: min 2, max 32
1. Select: Pause compute capacity after (5) consecutive minutes of inactivity
1. VPC: eksctl-zq-test-cluster/VPC ...
1. Subnet Group: aurora-zq-test
1. VPC SG: Choose existing
1. Existing VPC security groups: RDS_SG
1. Leave additional configuaration at defaults

## After the instance has been created

Try connecting with the following steps.

Add an ingress rule to RDS_SG for your current IP. 



### Install psql tools
If `psql --version` isn't there, then on MacOS:
```
brew update
brew install libpq
brew link --force libpq
```

### Shell for psql

```
kubectl run my-shell --rm -i --tty --image ubuntu -- bash
```
At the shell prompt of `#` type:
```
sudo apt-get update
sudo apt-get install postgresql-client
```
