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
helm repo add bitnami https://charts.bitnami.com/bitnami
```
### Create a Kubernetes cluster with Kind
In the k8s directory, there is a `kind-with-registry.sh` shell script to automate these steps. It creates a kind cluster that will connect to the local docker registry. It will create the docker registry if it does not already exist. This is based on the instructions here:
https://kind.sigs.k8s.io/docs/user/local-registry/

```
./kind-with-registry.sh
./zq-context.sh
```

This starts a container image within Docker to host a single-node Kubernetes cluster. 
We then create a 'zq' namespace in which to deploy our pods, and a corrresponding 'zq' context to set the zq namespace as default for subsequent kubectl and helm commands.

After the Kind cluster has come up, check to see if its pods are running with:
```
kubectl get pod -A
```

### Build a zqd image with Docker
There is a Dockerfile in the root directory. This Dockerfile does a 2-stage build to produce a relatively small image containing the zqd binary. It is structured to cache intermediate results so it runs faster on subsequent builds.

The following command builds and labels the container image:
```
DOCKER_BUILDKIT=1 docker build --pull --rm \
  -t "zqd:latest" -t "localhost:5000/zqd:latest" .
```
Notive this adds a tags for loading the image into the local docker registry creaeted by kind-with-registry.sh. To load the image into the registry:
```
docker push "localhost:5000/zqd:latest"
```
This copies the image into the container that is running your single-node Kubernetes cluster for Kind. Once it is there, use an image PullPolicy:Never to insure that the local copy of the image is used in our Kind cluster. Later, for remote deployments, we use image pullPolicy:IfNotPresent.

### Deploy zqd into the local cluster with helm
The K8s deployment and and service yaml for zqd is pretty simple. We use Helm 3 to parameterize the differences between local and remote deploys. (Note that there are good alternatives to Helm for this.) Helm can conveniently uninstall or upgrade zqd.

The helm 3 chart is in k8s/charts/zqd. To install zqd with the helm chart:
```
helm upgrade zqd-test-1 charts/zqd --install
```
This will install the chart if it is not yet present, or upgrade it to latest if it is installed. (The helm chart adds a timestamp as an annotation to the deployment, so it will always restart the pods.)

You can confirm the pod has started with:
```
kubectl get pod
```
And view Helm installs with:
```
helm ls
```
### Testing connectivity to zqd
A simple test for the zqd is to send a /status request. This is used for the K8s liveness probe. Substitute the pod id into a port-forward command:

```
kubectl port-forward zqd-test-1-66b5f9dc8-8dshh 9867:9867
```
And in another console, use curl:
```
curl -v http://localhost:9867/status
```
You should get an 'ok' response.

### Using zqd with zapi

This is a walk-through of a local test we  do before testing on our k8s cluster. This is outside the k8s cluster to get familiar with trouble-shooting.

As a prerequisite, you must have an S3 bucket and directory available for zqd. In the S3 console, or at the command line, create a 'datadir' for use with zqd. This directory will hold metadata for the spaces you will create with zapi.

You will need AWS credentials. On your local machine you can use `aws configure` and then you must set the environment variable, `AWS_SDK_LOAD_CONFIG=true`. Within Kubernetes, we will need to handle AWS credentials with secrets. More on that later.

Here is an example of creating a datadir, using an s3 bucket called `brim-scratch` in a directory called `mark` (change the s3 buckets for your setup.)
```
aws s3 ls # make sure the bucket exists!
zqd listen -data s3://brim-scratch/mark/zqd-meta
```
zqd will stay running in that console, listening at `localhost:9867` by default.
zqd will not create any s3 objects in zqd-meta until we issue a zapi command. Before using zapi, we use zar in another console to get sample data from our zq repo into s3:
```
zar import -R s3://brim-scratch/mark/sample-http-zng zq-sample-data/zng/http.zng.gz
```
This creates zng files in an s3 directory called `sample-http-zng` that we will use from zapi. To check what zar created:
```
aws s3 ls brim-scratch/mark --recursive
```
Now use zapi to make the zar data available to your running zqd instance. Note that zapi defaults to `localhost:9867` for zqd.
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space-1
```
The `-d` parameter provides the same s3 dir that we used in the `-R` parameter of the zar command above. This command creates a new space for zqd called `http-space-1`. If we again list the s3 directories with `aws s3 ls brim-scratch/mark --recursive` we will see that zqd has now created a new object under its `-datadir` which has a name generated from "sp_1g0" followed by entropy.

Now that you have a zqd running with a space, and you have made an archive from zar data, you can use zapi commands to query the sample data. Example:
```
zapi -s http-space-1 get "head 1"
zapi -s http-space-1 get "tail 1"
```
The zapi commands in quotes are juthe same as Brim queries. Notice that "head 1" runs much faster than "tail 1", which has to read more data from s3.

### Using zqd in the Kind cluster

#### AWS credentials
To access AWS S3 from zqd running as a Kubernetes pod, you must have AWS credentials available to the code running in the pod. We use K8s secrets to provide the credential to the deployment as env vars. The secrets are references in the Helm template deployment.yaml. The shell script `aws-credentials.sh` reads your credentials using `aws configure` and creates a K8s secret in the Kind cluster. (We will do something different for AWS EKS, because we can use IAM when the cluster is deployed in AWS.)

Before deploying zqd with helm, run:
```
./k8s/aws-credentials.sh
```
Then is you type:
```
kubectl get secret
kebctl get secret aws-credentials -oyaml
```
You will see the new objects. The secrets are base64 encoded.

#### Redeploy with helm
This step is not mandatory, but if you make a change in the helm templates, you generally will want to uninstall/reinstall the zqd.
```
helm uninstall zqd-test-1
helm install zqd-test-1 charts/zqd
```
To check if the AWS env vars are present in the deployment, these commands are helpful:
```
kubectl get deploy
kubectl describe deploy zqd-test-1
kubectl get deploy zqd-test-1 -oyaml
```

#### Redeploy zqd with an S3 datadir
When you deploy zqd, you specify a datadir like we did in the standalone example. At present, this datadir is specified in the deployment.yaml for the helm template and passed in as a parameter to the helm install.

In this example, we do a helm deploy with the same S3 datadir we used in the local example above:
```
helm uninstall zqd-test-1
helm install zqd-test-1 charts/zqd --set datadir="s3://brim-scratch/mark/zqd-meta"
```
Check the logs to see if zqd is running with the correct parameters:
```
stern zqd-test-1
```
Now follow the instructions that Helm printed out on install to port-forward for zapi:
```
export POD_NAME=$(kubectl get pods --namespace zq -l "app.kubernetes.io/name=zqd,app.kubernetes.io/instance=zqd-test-1" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace zq port-forward $POD_NAME 9867:9867
```

Now use zapi to create a space as in the local example above:
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space-1
```
And try some zapi queries:
```
zapi -s http-space-2 get "head 1"
zapi -s http-space-2 get "tail 1"
```
Notice that it is really slow now because it is running in resource-constrained local Kind cluster! :-)

We can also query http-space-2 with Brim, since it will connect to the same port-forward for zqd.

## Adding Observability

WIP!

Here we add several moving parts to out local K8s cluster that will allow us to measure zqd's performance and resource consumption.

First we will add a "service mesh" called Linkerd. 

https://linkerd.io/2/getting-started/

We do not need Linkerd for the typical use case of service meshes, which is to route intra-service messages, often gRPC, within a cluster. Instead, we leverage the side-car router as a conveninet way to monitor inbound and outbound traffic from ZQD. Conveniently, the Linkerd install also includes Prometheus and Grafana to support Linkerd's dashboard. We will configure these Prometheus and Grafana instances to also monitor zqd's Prometheus metrics.

We will also install the Kube State Metrics:

https://github.com/kubernetes/kube-state-metrics

That capture the Prometheus metrics published by Kubernetes. The combination of the Prometheus metrics from zqd, Linkerd, and KSM will provide us with thorough instrumention of I/O, CPU, and RAM use correlated in time with zqd queries.

### Step 1: install Linkerd into the Kind cluster

First insure that Linkerd command line tools are installed. Use the getting started guide, or on MacOS:
```
brew install linkerd
linkerd version
```
Make sure you are in the right context for your local Kind cluster, then:
```
linkerd install | kubectl apply -f -
```
Linkerd, when it is installed this way, will automatically inject sidecar containers into deployment than include Linkerd annotations. The various Linkerd services will run in their own 'linkerd' namespace and they take a while to start. To make sure it finishes the install, you can wait with:
```
kubectl wait --for=condition=available --timeout=120s -n linkerd deployment linkerd-tap
```


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

# Troubleshooting
## Shell into a K8s pod
```
kubectl get pod
```
Copy the pod id, thensub it into:
```
kubectl exec -it zqd-test-1-XXXXXXXX-99999 -- sh
```

## Problems with the Docker containers
Sometime a problem with the container will prevent it from starting in K8s, but you may be able to shell into the container in Docker to trouble-shoot. 
```
docker run -it zqd:latest sh
```
## Ports in K8s containers
In order for the kubelet liveness check to work, zqd must be listening on 0.0.0.0:9867. 
For `kubectl port-forward` to work, zqd must be listening on 127.0.0.1:9867.
To get this behavior, we set the command line flag `zqd listen -l :9867` -- so it will listen to the socket for all hosts.


# Tearing down the environment

Later, when you no longer need the cluster, you can remove it with:
```
kind delete cluster --name zq-local
```


