## Local development

Here are instructions for how to set up a local K8s cluster hosted on Kind (Kubernetes in Docker). These were tested on on a Macbook Pro. They can be adapted to Linux. In each section, we reference the setup instructions we used from other community projects and vendors. Those links have details on how to to do correponding setup on Linux.

At this point, zqd on K8s is designed to access log file hosts on Amazon S3. The K8s deployment on Kind assumes you will need AWS credentials to access S3. 

### Prerequisites

You need to install Docker on your dev machine. Install Docker Desktop:

https://hub.docker.com/editions/community/docker-ce-desktop-mac

In docker Settings/Resources, you will want to increase the default RAM and CPU allocations to Docker, depending on the size of the log files you want to test with.

Using Homebrew (https://brew.sh) install the following packages:
```
brew install kind stern helm
```
The Docker desktop install also installs a version of kubectl that works for this process. `kubectl version` should be 1.17 or later. If you want to update to the most current kubectl, then:
```
brew install kubernetes-cli
brew link --overwrite kubernetes-cli
```
Connect to this helm chart repo:
```
helm repo add bitnami https://charts.bitnami.com/bitnami
```
### Create a Kubernetes cluster with Kind
In the k8s directory, there is a `kind-with-registry.sh` shell script to automate these steps. It creates a kind cluster that will connect to the local docker registry. It will create the docker registry if it does not already exist. This is based on the instructions here:
https://kind.sigs.k8s.io/docs/user/local-registry/

```
./k8s/kind-with-registry.sh
./k8s/zq-context.sh
```

This starts a container image within Docker to host a single-node Kubernetes cluster. 
We then create a 'zq' namespace in which to deploy our pods, and a corrresponding 'zq' context to set the zq namespace as default for subsequent kubectl and helm commands.

After the Kind cluster has come up, check to see if its pods are running with:
```
kubectl get pod -A
```

## Delete the local cluster
When you no longer need the local cluster, you can remove it with:
```
kind delete cluster --name zq-local
```

