# Setup instructions for Temporal

Temporal in installed as a subchart of our zservice Helm "umbrella" chart.

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
Each use that will run Temporal must have an initialized database available. We use Aurora for the per-user Temporal databases. Setting up a Temporal database on the zq-test-aurora instance involve manual steps, because the Temporal database initialization and migration code is not designed to be run automated. (We confirmed this with the the Temporal dev team.)

## Database initialization for Temporal
We will follow a pattern similar to what we have done for setting up Aurora users for testing.

We will need a Docker container with the most recent build of `temporal-sql-tool`. Here's hopw to create that. Note that we are working with Temporal v1.7.0. For stability, we will not use other versions or try buildiong head from the repo.
```
git clone https://github.com/temporalio/temporal.git
git checkout v1.7.0
docker build -t temporal .
aws ecr get-login-password --region us-east-2 | \
  docker login --username AWS --password-stdin $ZQD_ECR_HOST/temporal
docker tag temporal $ZQD_ECR_HOST/temporal:1.7.0
docker push $ZQD_ECR_HOST/temporal:1.7.0
# launch K8s pod with this docker image
```


