# Deploying zqd on EKS

These are instructions for deploying a locally built image for zqd into an EKS cluster, in a namespace that is specific to your dev environment.

## First time setup

This assumes that you have a AWS IAM access on the AWS account that hosts the EKS cluster. You should have installed the AWS CLI.

https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html

You should also have Docker installed in order to build the zqd container image.

https://docs.docker.com/get-docker/

### Desktop tools

`kubectl`, `eksctl` and `helm` should be installed. See instructions at:

https://kubernetes.io/docs/tasks/tools/install-kubectl/

https://docs.aws.amazon.com/eks/latest/userguide/eksctl.html

https://helm.sh/docs/intro/install/

#### On MacOS
Using brew (https://brew.sh) for the install works well on MacOS:
```
brew tap weaveworks/tap
brew install weaveworks/tap/eksctl
brew install kubernetes-cli
brew link --overwrite kubernetes-cli
brew install helm
```

### EKS access

First connect to the EKS cluster so you have kubectl access. You or a colleague should follow the steps here to obtain cluster access:

https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html

#### Synopsis
The user who wants EKS cluster access, runs:
```
eksctl get cluster
eksctl utils write-kubeconfig --cluster zq-test
aws sts get-caller-identity
```
And passes on their user arn to the AKS admin, who adds it to the "MapUsers" list with:
```
kubectl edit -n kube-system configmap/aws-auth
```
`eksctl` can be used instead of kubectl to edit the aws-auth configmap. It is about the same amount of typing, e.g.
```
eksctl create iamidentitymapping --cluster zq-test --arn arn:aws:iam::123456:role/testing --group system:masters --username admin
```

### Environment variables for Makefile rules

In the zq Makefile, there are several rules to make the developer experience more consistent. These rules depend on having the following env vars defined:
```
export ZQD_ECR_HOST=123456789012.dkr.ecr.us-east-2.amazonaws.com
export ZQD_DATA_URI=s3://zqd-demo-1/mark/zqd-meta
export ZQD_K8S_USER=mark
export ZQD_TEST_CLUSTER=zq-test.us-east-2.eksctl.io
```
You must modify these to fit you environment. 
* ZQD_ECR_HOST is the host portion of the ECR service for your AWS account. 
* ZQD_DATA_URI is used to set the '-data' flag when zqd is started on Kubernetes. It should be an S3 bucket specific to you that you can use for testing. 
* ZQD_K8S_USER is your username, which is usually the same as your AWS IAM username.
* ZQD_TEST_CLUSTER is the host of the EKS cluster you are using.

### Create a K8s namespace for your development

Start by creating a namespace, specific to your user, in which to deploy zqd. The following Makefile rule does that. It is design to be run once. It can be run again if you have removed the namespace.
```
make kubectl-config
```

## Build and upload docker image
This development workflow assumes that you will build and test zq locally, then deploy it into your namespace in the EKS cluster. To build and push the zqd image to AWS ECR, use:
```
make docker-push-ecr
```
The tag on this image is based on `git describe` so it is specific to your branch. All zqd images share the same ECR repo.

## Install Postgres with Helm

Because the helm recipe for postgres uses a persistent volume claim to persist
the database between installs, we must create a kubernetes secret with postgres
passwords that will in kind persist between installs. Run this script to create
the secret with randomly generated passwords for the postgres admin and zqd user
accounts:

```
./k8s/postgres-secret.sh
```

The postgres database can now be installed via helm:

```
make helm-install-postgres
```


## Install with Helm
Helm is used to deploy the zqd image. Use:
```
make helm-install
```
To run Helm with the correct command line flags.

After the helm install, you can check the status of your install with:
```
helm ls
```
If you want to redeploy in your namespace, first uninstall the zqd instance with:
```
helm uninstall zqd
```
To check the status of your running pod in your namespace, use:
```
kubectl get pod
```
This will show the unique name of your running zqd pod. Copy that name for the following troubleshooting steps. If the status of the pod is 'Error' or 'ImagePullBackoff' (or something else not good), then you can get details with:
```
kubectl describe pod zqd-56b46985fc-bqv87
kubectl logs -c zqd zqd-56b46985fc-bqv87 -p -f
```
Edit the commands to use your pod name.

## Expose the endpoint for local development
Run the following shell script to expose the zqd endpoint on your local host:
```
./k8s/zqd-port.sh
```
This script kills existing port-forwards and creates a new port-forward. It also starts the Linkerd dashboard so you can monitor your endpoint.

## Quick test for zqd connectivity
Use zapi to create a Brim space if your ZQD_DATA_URI does not already contain a space, e.g.
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space
```
And try some zapi queries:
```
zapi -s http-space get -t "head 1"
zapi -s http-space get -t "tail 1"
```

You can also query http-space with Brim, since it will connect to the same port-forward for zqd that zapi uses.
