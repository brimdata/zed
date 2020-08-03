# Deploying the ZQ daemon in a Kubernetes cluster

The describes a procedure for deploying the ZQ daemon that you can connect to remotely with Brim. This is useful for when you are running Brim on a machine that need to access large log files that are in the data center where you are running the ZQ daemon.

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
Later, when you no longer need the cluster, you can remove it with:
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
The "killer feature" of Kind is that is makes it vary easy to copy container images into the local cluster without needing an image repo. To load the image into you Kind cluster:
```
kind load docker-image --name zq-local zqd:latest
```
This copies the image into the VM that is running your single-node Kubernetes cluster for Kind. Once it is there, use an image PullPolicy:Never to insure that the local copy of the image is used in our Kind cluster. Later, for remote deployments, we use image pullPolicy:IfNotPresent.

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
