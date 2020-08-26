# Deploying the ZQ daemon in a Kubernetes cluster

This describes a procedure for deploying the ZQD service that you can connect to remotely with Brim. This is useful for when you are running Brim on a machine that needs to access large log files that are in the data center where you are running the ZQD service.

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

### Build a zqd image with Docker
There is a Dockerfile in the root directory. This Dockerfile does a 2-stage build to produce a relatively small image containing the zqd binary. It is structured to cache intermediate results so it runs faster on subsequent builds.

The Makefile has a target `make docker` that creates a docker image with the correct version number passed in as LDFLAGS. `make docker` is the preferred way to build a docker image.

The `make docker` target also copies the image into a Docker registry that is accessed by your single-node Kubernetes cluster for Kind.

### AWS credentials
To access AWS S3 from zqd running as a Kubernetes pod, you must have AWS credentials available to the code running in the pod. We use K8s secrets to provide the credential to the deployment as env vars. The secrets are references in the Helm template deployment.yaml. The shell script `aws-credentials.sh` reads your credentials using `aws configure` and creates a K8s secret in the Kind cluster. (We will do something different for AWS EKS, because we can use IAM when the cluster is deployed in AWS.)

Run:
```
./k8s/aws-credentials.sh
```
Then try:
```
kubectl get secret
kubectl get secret aws-credentials -oyaml
```
You will see the new objects. The secrets are base64 encoded.

### Deploy zqd with a S3 datauri
In this example, we do a helm deploy that sets the S3 datauri for zqd. You should already have an S3 bucket set up for this. You can use any naming convention you want for your S3 datauri. In this example, the S3 bucket has a directory "mark" with a subdir call "zqd-meta". Change both of these values for your S3 setup. 
```
helm install zqd-test-1 charts/zqd --set datauri="s3://brim-scratch/mark/zqd-meta"
```
Check the logs to see if zqd is running with the correct parameters:
```
stern zqd-test-1
```
Now follow the instructions that Helm printed out on install to port-forward 9867:
```
export POD_NAME=$(kubectl get pods --namespace zq -l "app.kubernetes.io/name=zqd,app.kubernetes.io/instance=zqd-test-1" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace zq port-forward $POD_NAME 9867:9867
```

Before using zapi to access the running zqd, we use `zar import` in another console to copy sample data from our zq repo into s3. Change the directory name to match your s3 bucket.
```
zar import -R s3://brim-scratch/mark/sample-http-zng zq-sample-data/zng/http.zng.gz
```
This creates zng files in an s3 directory called `sample-http-zng` that we will use from zapi. To check what zar created:
```
aws s3 ls brim-scratch/mark --recursive
```

Now use zapi to create a space, just like the local example above:
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space-2
```
And try some zapi queries:
```
zapi -s http-space-2 get "head 1"
zapi -s http-space-2 get "tail 1"
```
Notice that it is really slow now because it is running in resource-constrained local Kind cluster! :-)

We can also query http-space-2 with Brim, since it will connect to the same port-forward for zqd.

## Adding Observability

Here we add several moving parts to our local K8s cluster that will allow us to measure zqd's performance and resource consumption.

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

### Step 2: Redeploy zqd to get Linkerd sidecar proxy injection

The zqd from the section "Redeploy zqd with an S3 datauri" will not yet have a Linkerd proxy. You can check this by doing this -- sub the pod id you get from `get pod` into the `describe pod`
```
kubectl get pod
kubectl describe pod zqd-test-1-99999999-XXXXX
```
You will see that the pod has one container called `zqd`.
After installing Linkerd, reinstall with helm like you did before:
```
helm uninstall zqd-test-1
helm install zqd-test-1 charts/zqd --set datauri="s3://brim-scratch/mark/zqd-meta"
```
Now repeat the `describe pod` and you will see that the pod has a second container called `linkerd-proxy`. This proxy monitors both inbound and outbound traffic.

### Step 3: Use the Linkerd dashboard to monitor zqd

This part assumes you have a space created for zqd, as described above. Start the Linkerd dashboard with:
```
linkerd dashboard &
```
On most desktops, this will start a browser window with the dashboard. On the home screen of the dashboard under HTTP metrics you should see your namespace, zq. There is a grafana icon in the far rigght column next to zq. Click on that, and you will see some basic metrics for traffic through zqd.

### Step 4: A Grafana dashboard for zqd
Grafana is also running in the linkerd namespace: it serves the Linkerd dashboard. Now we will add our own Grafana dashboard for zqd.

This assumes that you have run `linkerd dashboard &` and that it has port-forwarded the Grafana instance to `localhost:50750`. Use a web browser to connect to the Grafana instance running in the linkerd nampespace:
```
http://localhost:50750/grafana
```
`k8s/grafana-dashboard.json` is a sample Grafana dashboard for zqd. To import the dashboard into the local Grafana instance, click on the "+" sign on left hand side of the Grafana home screen. From the drop=down that appears, select "import". On the import screen, select upload .json file, and choose `k8s/grafana-dashboard.json`. (You may have to change the name or uid on the import screen.)

To experiment with the dashboard, you may want to place some repetitive load on the zqd instance. There is a shell script `k8s/zapi-loop.sh` you can use to do a simple zq query in a loop while you watch the dashboard. This script assumes you have completed "Deploy zqd with a S3 datauri" above.

#### Dashboard details
The sample dashboard that includes the following PromQL queries to monitor the zqd container:
```
container_memory_usage_bytes{id=~"/kube.+",container="zqd"}
rate(container_cpu_usage_seconds_total{id=~"/kube.+",container="zqd"}[1m])
sum(rate(tcp_read_bytes_total{app_kubernetes_io_name="zqd"}[1m]))
sum(rate(container_network_receive_bytes_total[1m]))
```
The linkerd metrics for the queries above are described in:
https://linkerd.io/2/reference/proxy-metrics/

The cAdvisor metrics for the queries above are described in:
https://github.com/google/cadvisor/blob/master/docs/storage/prometheus.md

## Deploying the ZQ daemon to an AWS EKS cluster

This walks through a procedure for setting up an EKS cluster on an AWS account. If you are contributing to ZQ, keep in mind the the AWS resources allocated here can be expensive. If you are working through this to learn, be sure to tear everything down ASAP after running experiments.

We reference AWS docs freqently. The initial process is derived from:

https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html

The commands we used in testing are detailed below.

We used the AWS CLI version 2. Install intructions are here:

https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html

### Creating the EKS Cluster
Choose a region for the cluster. For the examples below, we used region us-east-1. (Hint: this is because us-east-1 is the lowest cost region for some S3 charges.) We set a default region with `aws configure` so it is not included in the CLI commands.

We use AWS `eksctl` to create the cluster. To install eksctl on MacOS:
```
brew tap weaveworks/tap
brew install weaveworks/tap/eksctl
eksctl version
```
Then create the cluster:
```
eksctl create cluster -f k8s/cluster.yaml
```
This command usually takes at least 10 minutes to complete. It uses parameters provided in `k8s/cluster.yaml` -- you can edit this file to change it as needed.

### Creating a zqeks namespace and a kubectl config context

This creates a context for the EKS cluster. You need to change the user and cluster in the example below. Use `kubectl config get-contexts` to get the values for your user and cluster.
```
kubectl create namespace zqdev
kubectl config set-context zqdev \
  --namespace=zqdev \
  --cluster=zq-dev.us-east-2.eksctl.io \
  --user=mark@zq-dev.us-east-2.eksctl.io
kubectl config use-context zqdev
```

### Install Linkerd into the EKS cluster

Insure that Linkerd command line tools are installed (see local install above).
```
linkerd install | kubectl apply -f -
```

### Use zar import to get some data into an S3 bucket in the region
We will use zar to create an S3 bucket in this region with sample data. 

If you do not already have a bucket you want to use in the same region as your EKS cluster, create it with `aws s3 mb`, e.g.
```
aws s3 mb s3://zqd-demo-1
```
Now use zar to create an object in S3. 

For the following zar import, change the directory name to match your s3 bucket.
```
zar import -R s3://zqd-demo-1/mark/sample-http-zng zq-sample-data/zng/http.zng.gz
```
This creates zng files in an s3 directory called `sample-http-zng` that we will use from zapi after zqd has been deployed. To check what zar created:
```
aws s3 ls zqd-demo-1/mark --recursive
```

### Pushing locally built Docker images from your local dev machine to ECR
You can push locally built Docker images to AWS ECR for deployment on EKS. These instructions are derived from:

https://docs.aws.amazon.com/AmazonECR/latest/userguide/getting-started-cli.html

Create an ECR repo for zqd:

```
aws ecr create-repository \
    --image-scanning-configuration scanOnPush=true \
    --repository-name zqd
```

When the repo is created at the command line it returns JSON that looks like:
```
"repositoryUri": "123456789012.dkr.ecr.us-east-1.amazonaws.com/gateway"
```
Sustitute this URI into the tag, login and push steps. To push the zqd image created with the local build:
```
make docker
docker tag zqd 123456789012.dkr.ecr.us-east-1.amazonaws.com/zqd
aws ecr get-login-password --region us-east-1 | docker login \
  --username AWS --password-stdin 123456789012.dkr.ecr.us-east-1.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-1.amazonaws.com/zqd
```

### Deploy the zqd service with Helm
Here is an example Helm install. This assumes you created an S3 resident space as in the local install above. This is similar to the local Helm deploy. It uses the same charts, but command line parameters overide the Values.yaml to provide specific configuration for EKS.

Substitute the image.repository you created above. Note that unlike the local deployment, we do not use K8s secrets for AWS credentials because the `cluster.yaml` above specified IAM policies for S3 access.

```
helm install zqd-test-1 charts/zqd \
  --set AWSRegion="us-east-2" \
  --set datauri="s3://zqd-demo/mark/zqd-meta" \
  --set image.repository="123456789012.dkr.ecr.us-east-1.amazonaws.com/" \
  --set useCredSecret=false
```

### Exposing the zqd endpoint

Work in progess...

For now just use:
```
export POD_NAME=$(kubectl get pods --namespace zq -l "app.kubernetes.io/name=zqd,app.kubernetes.io/instance=zqd-test-1" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace zq port-forward $POD_NAME 9867:9867
```

## Setting up an EC2 instance to run zar import

This is for importing large zeek logs from S3 to a zar archive also on S3.

Create an EC2 instance in the same region as the S3 buckets you want to access. SSH into the instance, and:
```
# provide your credentials
aws configure
# make sure you can see the S3 buckets
aws s3 ls
# install golang tools
wget https://golang.org/dl/go1.14.7.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.14.7.linux-arm64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bash_profile
source ~/.bash_profile
go version  # make sure go is there!
# install git, clone and install zq
sudo yum install git -y
git clone https://github.com/brimsec/zq
make install
~/go/bin/zar help  # make sure zar is built!
```
Now you have an environment where you can run zar and it can operate on data in S3 with good bandwidth. Example zar command:
```
~/go/bin/zar import -R s3://zqd-demo-1/mark/zeek-logs/dns s3://brim-sampledata/wrccdc/zeek-logs/dns.log.gz
```

## [Trouble shooting](trouble-shooting.md)

## Tearing down the environment

Later, when you no longer need the local cluster, you can remove it with:
```
kind delete cluster --name zq-local
```
