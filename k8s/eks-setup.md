# Setting up an EKS cluster for Brim development

We walk through a procedure for setting up an EKS cluster on an AWS account. If you are contributing to zq, keep in mind the the AWS resources allocated here can be expensive. If you are working through this to learn, be sure to tear everything down ASAP after running experiments.

We reference AWS docs freqently. The initial process is derived from:

https://docs.aws.amazon.com/eks/latest/userguide/getting-started-eksctl.html

The commands we used in testing are detailed below.

We used the AWS CLI version 2. Install intructions are here:

https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html

Choose a region for the cluster. For the examples below, we used region us-east-2. We set a default region with `aws configure` so it is not included in the CLI commands.

## Creating the EKS Cluster

We use AWS `eksctl` to create the cluster. To install eksctl on MacOS:
```
brew tap weaveworks/tap
brew install weaveworks/tap/eksctl
eksctl version
```
Then create the cluster. You may wish to edit `k8s/cluster.yaml` to choose your own cluster name and other options such as node instance type.
```
eksctl create cluster -f k8s/cluster.yaml
```
This command usually takes at least 10 minutes to complete. It uses parameters provided in `k8s/cluster.yaml` -- you can edit this file to change it as needed.

## Creating a zqeks namespace and a kubectl config context

This creates a context for the EKS cluster. You need to change the user and cluster in the example below. Use `kubectl config get-contexts` to get the values for your user and cluster.
```
kubectl create namespace zqtest
kubectl config set-context zqtest \
  --namespace=zqtest \
  --cluster=zq-test.us-east-2.eksctl.io \
  --user=mark@zq-test.us-east-2.eksctl.io
kubectl config use-context zqtest
```

# Setting up an EKS cluster, optional steps

The following optional steps enable the following capablities:
* Use cluster autoscaling on EKS
* Install a Linkerd service mesh with sidecar proxies for observability
* Uses Prometheus and Grafana to monitor zqd

## Enabling the EKS Cluster Autoscaler
Instructions based on:
https://docs.aws.amazon.com/eks/latest/userguide/cluster-autoscaler.html

Use the following kubectl commands:
```
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml
kubectl -n kube-system annotate deployment.apps/cluster-autoscaler cluster-autoscaler.kubernetes.io/safe-to-evict="false"
kubectl -n kube-system edit deployment.apps/cluster-autoscaler
```
The edit command open a file, look for the line:
```
        - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/<YOUR CLUSTER NAME>
```
And change it to be:
```
        - --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/zq-test
        - --balance-similar-node-groups
        - --skip-nodes-with-system-pods=false
```
Finally, as of 9/21/2020, set the image to release 1.17.3:
```
kubectl -n kube-system set image deployment.apps/cluster-autoscaler cluster-autoscaler=us.gcr.io/k8s-artifacts-prod/autoscaling/cluster-autoscaler:v1.17.3
```

## Install Linkerd into the EKS cluster

Instructions based on: https://linkerd.io/2/getting-started/

Insure that Linkerd command line tools are installed 
```
brew install linkerd
linkerd version
```
Then use kubectl to deploy Linkerd with:
```
linkerd install | kubectl apply -f -
```

## Tear-down
Using the AWS console is the most convenient way to delete your EKS cluster, since you will want to double-check to make sure it is not running after you are done.

When you want to delete the EKS cluster, you must first delete the nodegroup. This will take a few minutes.

# Appendix: tasks that are automated by Tilt but that are useful to remember

## Pushing locally built Docker images from your local dev machine to ECR
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
"repositoryUri": "123456789012.dkr.ecr.us-east-2.amazonaws.com/gateway"
```
Sustitute this URI into the tag, login and push steps. To push the zqd image created with the local build:
```
make docker
docker tag zqd 123456789012.dkr.ecr.us-east-2.amazonaws.com/zqd
aws ecr get-login-password --region us-east-2 | docker login \
  --username AWS --password-stdin 123456789012.dkr.ecr.us-east-2.amazonaws.com
docker push 123456789012.dkr.ecr.us-east-2.amazonaws.com/zqd
```
## Use zar import to get some data into an S3 bucket in the region
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

## Deploy the zqd service with Helm
Here is an example Helm install. This assumes you created an S3 resident space as in the local install above. This is similar to the local Helm deploy. It uses the same charts, but command line parameters overide the Values.yaml to provide specific configuration for EKS.

Substitute the image.repository you created above. Note that unlike the local deployment, we do not use K8s secrets for AWS credentials because the `cluster.yaml` above specified IAM policies for S3 access.

```
helm install zqd-test-1 charts/zqd \
  --set AWSRegion="us-east-2" \
  --set datauri="s3://zqd-demo/mark/zqd-meta" \
  --set image.repository="123456789012.dkr.ecr.us-east-2.amazonaws.com/" \
  --set useCredSecret=false
```

## Exposing the zqd endpoint

To access the running zqd instance, use kubectl port-forward:
```
kubectl port-forward svc/zqd-test-1 9867:9867 &
```
There is a script `k8s/ports.sh` that does this after removing any existing port-forwards.

## Using a K8s hosted zqd to search large logs in the cloud
This decribes a set of steps to use `zar import` to process large log files that can then be searched with the zqd you just deployed. In order to do this, you will need an EC2 instance in the same region as your EKS cluster. Follow the steps below to set this up.

## Setting up an EC2 instance to run zar import

Create an EC2 instance in the same region as the S3 buckets you want to access. SSH into the instance, and:
```
# provide your credentials
aws configure
# make sure you can see the S3 buckets
aws s3 ls
# install golang tools from https://golang.org/dl/
wget https://golang.org/dl/go1.14.7.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.14.7.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bash_profile
source ~/.bash_profile
go version  # make sure go is there!
# install git, clone and install zq
sudo yum install git -y
git clone https://github.com/brimsec/zq
cd zq
make install
~/go/bin/zar help  # make sure zar is built!
```
Now you have an environment where you can run zar and it can operate on data in S3 with good bandwidth. You can update it as needed with:
```
cd zq
git pull
make install
```
You may want to use an SSD for $TMPDIR for zar import where the sort can spill to disk. Consider this if zar runs out of space for sorting.

First use `lsblk` to find an SSD device, then mount it similarly to the following example:
```
lsblk  # look for a free device, e.g. nvme1n1
sudo mkfs -t xfs /dev/nvme1n1
sudo mkdir /data
sudo mount /dev/nvme1n1 /data
df  # Verify that /data is mounted
sudo chown ec2-user /data  # so zar can write to it
echo "export TMPDIR=/data" >> ~/.bash_profile
source ~/.bash_profile
```

## Using zar import for large S3 logs
In the following examples, we import some log files that are already present in S3 in zeek format, and create a local archive for use by zqd. The following commands should be run on an ec2 instance in the same region as your EKS cluster:
```
# this is a 7 MB log -- takes a few seconds
time ~/go/bin/zar import -R s3://zqd-demo-1/mark/zeek-logs/dpd s3://brim-sampledata/wrccdc/zeek-logs/dpd.log.gz

# and a 267 MB log -- takes a few minutes
time ~/go/bin/zar import -R s3://zqd-demo-1/mark/zeek-logs/dns s3://brim-sampledata/wrccdc/zeek-logs/dns.log.gz

# and a 14 GB log -- takes about two hours, and will fail without TMPDIR space
time ~/go/bin/zar import -R s3://zqd-demo-1/mark/zeek-logs/conn s3://brim-sampledata/wrccdc/zeek-logs/conn.log.gz
```
Assuming you have an EKS cluster set up as described above, with the port-forward in effect,  you can use zapi to create "spaces" for Brim:
```
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/dpd dpd-space
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/dns dns-space
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/conn conn-space
```

Now you can run Brim, and it will use the same local:9867 port used by zapi. You will see the three spaces you just created, dpd-space, dns-space, and conn-space.

## Observing resource usage in Grafana

You can run `linkerd dashboard &` to get to a Grafana dashboard for zqd. `k8s/ports.sh` also includes `linkerd dashboard &`.

See above, at "Step 4: A Grafana dashboard for zqd". Those instructions work the same for a remote Grafana dashboard.

