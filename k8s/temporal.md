# Setup instructions for Temporal

Temporal is installed as a subchart of our zservice Helm "umbrella" chart.

Our Helm subchart is based on:
https://github.com/temporalio/helm-charts
With minimal changes. The installation pattern we follow is for 
We use a minimal install of Temporal, based on the instructions at:

https://github.com/temporalio/helm-charts#install-with-your-own-postgresql

We supress the install of Grafana, Prometheus, and Kafka by including the following flags:
```
    --set prometheus.enabled=false \
    --set grafana.enabled=false \
    --set kafka.enabled=false \
```
Each user that will run Temporal must have an initialized database available. We use Aurora for the per-user Temporal databases. Setting up a Temporal database on the zq-test-aurora instance involve manual steps, because the Temporal database initialization and migration code is not designed to be run automated. (We confirmed this with the the Temporal dev team.)

## Database initialization for Temporal
We will follow a pattern similar to what we have done for setting up Aurora users for testing.

We will need a Docker container with the most recent build of `temporal-sql-tool`. Here's hopw to create that. Note that we are working with Temporal v1.7.0. For stability, we will not use other versions or try buildiong head from the repo.
```
# The git clone and the docker build steps can be skipped
# if an image has already been pushed to $ZQD_ECR_HOST/temporal:1.7.0
git clone https://github.com/temporalio/temporal.git
git checkout v1.7.0
docker build -t temporal .
aws ecr get-login-password --region us-east-2 | \
  docker login --username AWS --password-stdin $ZQD_ECR_HOST/temporal
docker tag temporal $ZQD_ECR_HOST/temporal:1.7.0
docker push $ZQD_ECR_HOST/temporal:1.7.0
# launch K8s pod with this docker image
kubectl apply -f k8s/temporal-sql-job.yaml
kubectl get pod
# use the pod name from get pod in the command below
kubectl exec --stdin --tty temporalsql-XX999 -- sh
```
In the shell for the pod, use these commands, adapted for your username, to set up the Temporal databases. Note that every database name must be qualified by prefixing a username, because we need seperate Temporal databases for test isolation.

```
export SQL_PLUGIN=postgres
export SQL_HOST=<RDS host>
export SQL_PORT=5432
export SQL_USER=theusername
export SQL_PASSWORD=thepassword

temporal-sql-tool create-database -database theusername_temporal
SQL_DATABASE=theusername_temporal temporal-sql-tool setup-schema -v 0.0
SQL_DATABASE=theusername_temporal temporal-sql-tool update -schema-dir schema/postgresql/v96/temporal/versioned

temporal-sql-tool create-database -database theusername_temporal_visibility
SQL_DATABASE=theusername_temporal_visibility temporal-sql-tool setup-schema -v 0.0
SQL_DATABASE=theusername_temporal_visibility temporal-sql-tool update -schema-dir schema/postgresql/v96/visibility/versioned
```
Afterwards you can delete the temporalsql pod.

## Helm chart for Temporal

https://github.com/temporalio/helm-charts

Includes a helm chart for Temporal, which includes dependencies on:
Cassandra, ElasticSearch, Kafka (with Zookeeper), Promethueus, Grafana
We use the helm chart following a pattern similar to:
https://github.com/temporalio/helm-charts#install-with-your-own-postgresql
And avoid all the dependencies above.

temporalio/helm-charts above is not designed to be used as a subchart within an umbrella Helm chart, and ths is not available in a helm repository. Brim wants to use it as a subchart, so we have followed a strategy similar to what is described here:
https://medium.com/@mattiaperi/create-a-public-helm-chart-repository-with-github-pages-49b180dbb417
In order to create a respository that we can use for our subchart dependency on Temporal.
We have created a git repo:
https://github.com/brimsec/helm-chart
That includes the repo `https://github.com/temporalio/helm-charts` as a git submodule.
The steps to do this, in case they need to be repeated, are:
```
git clone https://github.com/brimsec/helm-chart.git
cd helm-chart/helm-chart-sources
git submodule add https://github.com/temporalio/helm-charts.git temporal
cd temporal
git checkout v1.7.0
helm dependency update
cd ../..
helm package helm-chart-sources/*
git status # should have untracked file: temporal-0.2.2.tgz
helm repo index --url https://brimsec.github.io/helm-chart/ .
# repo index creates the index.yaml file that is needed to "publish" the chart
git add .
git commit -m "new index"
git push
```
Assuming that all the steps from Mattia Peri's article above have been taken, the Temporal helm chart will be available from the brim repo:
```
helm repo add brim https://brimsec.github.io/helm-chart/
helm search repo temporal
```
Will show the newly added Temporal helm chart.

## Using Makefile rule to deploy temporal
Temporal may be deployed as part of the zservice Helm chart. Use:
```
make helm-install-with-aurora-temporal
```
Prior to using this target, you must set three env vars that are used to configure Temporal DB access:
```
ZQD_AURORA_USER=theusername
ZQD_AURORA_PW=$$(kubectl get secret postgres --template="{{ index .data \"postgresql-password\" }}" | base64 --decode)
ZQD_AURORA_HOST=$(aws rds describe-db-cluster-endpoints \
		--db-cluster-identifier zq-test-aurora \
		--output text --query "DBClusterEndpoints[?EndpointType=='WRITER'] | [0].Endpoint"):5432
```


