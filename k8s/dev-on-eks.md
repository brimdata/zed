# Deploying zqd on EKS

These are instructions for deploying oa locally built image for zqd into an EKS cluster, in a namespace that is specific for your development.

## First time setup

First connect to the EKS cluster so you have kubectl access. You or a collegue should follow the steps here to obtain cluster access:

https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html

In the zq Makefile, there are several rules to make the developer experience more consistent. These rules depend on having the following env vars defined:
```
export ZQD_ECR_HOST=123456789012.dkr.ecr.us-east-2.amazonaws.com
export ZQD_DATA_URI=s3://zqd-demo-1/mark/zqd-meta
export ZQD_K8S_USER=mark
export ZQD_TEST_CLUSTER=zq-test.us-east-2.eksctl.io
```
You must modify these to fit you environment. ZQD_ECR_HOST is the host portion of the ECR service for your AWS account. ZQD_DATA_URI is used to set the '-data' flag when ZQD is started on Kubernetes. It should be an S3 bucket specific o you that you can use for testing. ZQD_K8S_USER is your username, which is usually the same as you AWS IAM username. ZQD_TEST_CLUSTER is the host of the EKS cluster you are using.

The Makefile rules are docker and kubectl commands that use these env vars.

Start by creating a namespace, specific to you user, in which to deploy zqd. The following Makefile rule does that. It is design to be run once, and can be run again if you have removed the namespace.
```
make kubectl-config
```

## Build and upload docker image
This development workflow assumes that you will build and test zq locally, then deploy it into your namespace in the EKS cluster. To build and push the zqd image to AWS ECR, use:
```
make docker-push-ecr
```
The tag on this image is based on `git describe` so it is specific to your branch. All zqd images are assumed to share the same ECR repo.

## Install image
Helm is used to deploy the zqd image. Use:
```
make helm-install
```
To run Helm with the correct command line flags.

After helm-install, you can check the status of your install with:
```
helm ls
```
If you want to redeploy in you test env, first uninstall the zqd instance with:
```
helm uninstall zqd
```
To check the status of your running pod in your namespace, use:
```
kubectl get pod
```
To see the unique name of your running zqd pod. Copy that name for the following troubleshooting steps. If the status of the pod in 'Error' of 'ImagePullBackoff' (or something else not good), then you can get details with:
```
kubectl describe pod zqd-56b46985fc-bqv87
kubectl logs zqd-56b46985fc-bqv87 -p
```
Edit the commands to use your pod name.

## Expose the endpoint for local development
Run the following shell script to expose the zqd endpoint on you local host:
```
./k8s/ports.sh
```
This script kills existing port-forwards and creates a new port-forward. It also starts the Linkerd dashboard so you can monitor your endpoint.

## Quick test for zqd connectivity
Use zapi to create a Brim space if your ZQD_DATA_URI does not already contain a space, e.g.
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space
```
And try some zapi queries:
```
zapi -s http-space get "head 1"
zapi -s http-space get "tail 1"
```

You can also query http-space with Brim, since it will connect to the same port-forward for zqd that zapi uses.

