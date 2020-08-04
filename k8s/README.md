# Deploying the ZQ daemon in a Kubernetes cluster

This describes a procedure for deploying the ZQ daemon that you can connect to remotely with Brim. This is useful for when you are running Brim on a machine that need to access large log files that are in the data center where you are running the ZQ daemon.

Currently we support zqd access to ZAR files stored on Amazon S3, so we also describe a procedure for deploying zqd to a K8s cluster hosted on AWS EKS.

## Local development

For convenience of development, here is a process to deploy zqd in a local K8s cluster hosted on Kind (Kubernetes in Docker.) As a prerequsite, you should have installed Docker on your dev machine. At the moment, I am developing on a Macbook Pro, so I have detailed instructions for MacOS. These can be adapted to Linux.

### MacOs instructions

Install Docker Desktop:

https://hub.docker.com/editions/community/docker-ce-desktop-mac

In docker Settings/Resources, you will want to increase the default RAM and CPU allocations to Docker, depending on the size of the log files you want to test with.

Using Homebrew (https://brew.sh) install the following packages:
```
brew install kind stern helm
```
The Docker desktop install also installed a version of kubectl that works for this process. If you want to update to the most current kubectl, then:
```
brew install kubernetes-cli
brew link --overwrite kubernetes-cli
```
Connect to this helm chart repo:
```
helm repo add stable https://kubernetes-charts.storage.googleapis.com/
```

### Create a Kubernetes cluster with Kind
In the k8s directory, there is a `create-cluster.sh` shell script to automate these steps. The manual procedure is described here.
```
kind create cluster --name zq-local
```
This starts a container image within Docker to host a single-node Kubernetes cluster. Later, when you no longer need the cluster, you can remove it with:
```
kind delete cluster --name zq-local
```
After the Kind cluster has come up, check to see if its pods are running with:
```
kubectl get pod -A
```
We will create a 'zq' namespace in which to deploy our pods, and a corrresponding 'zq' context to set the zq namespace as default for subsequent kubectl and helm commands:
```
kubectl create namespace zq
kubectl config set-context zq \
  --namespace=zq \
  --cluster=kind-zq-local \
  --user=kind-zq-local
kubectl config use-context zq
```

### Build a zqd image with Docker
There is a Dockerfile in the root directory. This Dockefile does a 2-stage build to produce a relatively small image containing the zqd binary. It is structured to cache intermediate results so it runs faster on subsequent builds.

The following command builds and labels the container image:
```
docker build --pull --rm -t zqd:latest .
```
The "killer feature" of Kind is that it makes it very easy to copy container images into the local cluster without needing an image repo. To load the image into your Kind cluster:
```
kind load docker-image --name zq-local zqd:latest
```
This copies the image into the container that is running your single-node Kubernetes cluster for Kind. Once it is there, use an image PullPolicy:Never to insure that the local copy of the image is used in our Kind cluster. Later, for remote deployments, we use image pullPolicy:IfNotPresent.

### Deploy zqd into the local cluster with helm
The K8s deployment and and service yaml for zqd is pretty simple. We use Helm 3 to parameterize the differences between local and remote deploys. (Note that there are good alternatives to Helm for this.) Helm can conveniently uninstall or upgrade zqd.

The helm 3 chart is in k8s/charts/zqd. To install zqd with the helm chart:
```
helm install zqd-test-1 charts/zqd
```
You can confirm the pod has started with:
```
kubectl get pod
```
NOTE: this is WIP, and the pod does not work yet! It will stay in CrashLoopBackoff.

## WIP: Deploying the ZQ daemon to an AWS EKS cluster

NOTE: this EKS procedure is not yet working end-to-end in the k8s branch!

This walks through a procedure for setting up an EKS cluster on an AWS account. If you are contributing to ZQ, keep in mind the the AWS resources allocated here can be expensive. If you are working through this to learn, be sure to tear everything down ASAP after running experiments.

We reference AWS docs freqently. The initial process is derived from:
https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html
The commands we used in testing are detailed below.

We used the AWS CLI version 2. Install intructions are here:
https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html

You must choose a region for the cluster. For the examples below, we used region us-east-1. (Hint: this is because us-east-1 is the lowest cost region for some S3 charges.) We set a default region with `aws configure` so it is not included in the CLI commands.

Before starting, you will need to create a pem file for EKS to access EC2:
https://docs.aws.amazon.com/cli/latest/userguide/cli-services-ec2-keypairs.html#creating-a-key-pair
After creating the pem file, extract the public key. If you already have a preferred key pair, just extract the public key.
```
aws ec2 create-key-pair --key-name zqKeyPair --query 'KeyMaterial' --output text > zq-eks-test.pem
ssh-keygen -y -f zq-eks-test.pem > zq-eks-test.pub
```
We use AWS `eksctl` to create the cluster. To install eksctl on MacOS:
```
brew tap weaveworks/tap
brew install weaveworks/tap/eksctl
eksctl version
```
Then create the cluster:
```
eksctl create cluster \
--name zqtest \
--version 1.17 \
--nodegroup-name standard-workers \
--node-type t3.medium \
--nodes 1 \
--nodes-min 1 \
--nodes-max 3 \
--ssh-access \
--ssh-public-key zq-eks-test.pub \
--managed \
--asg-access
```
This command usually takes at least 10 minutes to complete.
In this example, we limit the cluster size to 3. This is enough to test autoscaling, but the max might need to be increased for searches in very large log files. If you keep the cluster up during dev and testing, you may want to manually scale it down with:
```
eksctl scale nodegroup --cluster test -m 1 -N 1 -M 1 -n standard-workers --region us-east-1
```
This decreases the maximum to 1, which is as low as you can go with an EKS cluster.
Later, when you want to delete the EKS cluster, you must first delete the nodegroup. This will take a few minutes. (BTW, I have found that deletes from the AWS console are at least as fast, and often faster, than using the AWS CLI to delete.)

### Creating a zqeks namespace and a kubectl config context

Similar to the create-cluster.sh script above, we create a context for the EKS cluster. You need to change the user and cluster in the example below. Use `kubectl config get-contexts` to get the values for your user and cluster.
```
kubectl create namespace zqeks
kubectl config set-context zqeks \
  --namespace=zqeks \
  --cluster=zqeks.us-east-2.eksctl.io \
  --user=mark@zqeks.us-east-2.eksctl.io
kubectl config use-context zqeks
```

## Tag and push images to ECR
This is how you can push locally built images to ECR. When there is a build pipeline for Docker images set up with Jenkins (or Gitlab, etc.), then it will handle this process. It is pretty typical to install Jenkins for image builds into the EKS dev cluster. Github and Gitlab have features that do this too, and they are relatively inexpensive. (E.g. https://docs.github.com/en/actions/creating-actions/creating-a-docker-container-action)

### Pushing locally Docker images from your local dev machine to ECR
You can push the Docker images build is the previos section to AWS ECR for deployment on EKS.

NOTE: Often this procedure is replaced by a CI/CD deployment pipeline.

Use this doc:
https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html

Create an ECR repo for zqd:

```
aws ecr create-repository \
    --image-scanning-configuration scanOnPush=true \
    --repository-name zqd
```

This must be run for each service, since in ECR terms, every image has it's own 'repo'. When the repo is created at the command line it returns JSON that looks like:
```
"repositoryUri": "123456789012.dkr.ecr.us-east-1.amazonaws.com/gateway"
```
Sustitute this URI into the tag, login and push steps. To push the zqd image created with the local build:
```
docker tag zqd 123456789012.dkr.ecr.us-east-1.amazonaws.com/zqd
aws ecr get-login-password --region us-east-1 \
 | docker login --username AWS \
   --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/zqd
```

### Exposing the zqd endpoint
You may will want to limit the exposure of your zqd endpoint, probably with a security group. (TBD)

### Using centralized logging
The default logging on K8s cluster just uses in-memory logs for the deployed pods. This is inconvenient for trouble-shooting. There are a number of centralized logging services available. The free verion of Papertrail (https://www.papertrail.com) is an easy place to start. If you create a free Papertrail account, the following instructions work for adding the log output from you pods in the EKS cluster to the Papertrail stream:
https://help.papertrailapp.com/kb/configuration/configuring-centralized-logging-from-kubernetes/
These are kubectl commands. Substitute your account ID for XXXXX
```
kubectl create secret generic papertrail-destination --from-literal=papertrail-destination=syslog+tls://logsN.papertrailapp.com:XXXXX
kubectl create -f https://help.papertrailapp.com/assets/files/papertrail-logspout-daemonset.yml
```







