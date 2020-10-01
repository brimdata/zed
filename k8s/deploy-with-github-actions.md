There are two jobs in `.github/workkflows/ci.yaml` that support automatic build and deploy of zqd into the Brim K8s test cluster on AWS EKS.

`ecr-image-push` uses the Makefile rule `make docker-push-ecr` to build a docker image and push it to the AWS ECR repo.

`eks-test-deploy` logs in to the test cluster with the user ci-master, and uses Helm to deploy the zqd image into the ci-master namespace.
