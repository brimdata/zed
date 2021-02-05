# Notes on creating AWS Aurora Postgres instance for test cluster

1. We used the AWS console
1. Choose:RDS Create Database
1. Select: Standard Create
1. Select: Amazon Aurora with PostgreSQL compatibility
1. Select: Serverless
1. DB cluster id: zq-test-db
1. Username: postgres
1. Password: Autogenerate
1. Capacity Settings: min 2, max 32
1. Select: Pause compute capacity after (5) consecutive minutes of inactivity
1. VPC: eksctl-zq-test-cluster/VPC ...
1. Select: Creat New DB Subnet Group
1. VPC SG: Choose existing (default)
1. Leave additional configuaration at defaults
