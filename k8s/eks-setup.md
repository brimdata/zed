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

# Appendix: tasks that are included in Makefile rules but that are useful to remember

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
There is a script `k8s/zqd-port.sh` that does this after removing any existing port-forwards.

## Observing resource usage in Grafana

You can run `linkerd dashboard &` to get to a Grafana dashboard for zqd. `k8s/linkerd-port.sh` also runs `linkerd dashboard &`.

See above, at "Step 4: A Grafana dashboard for zqd". Those instructions work the same for a remote Grafana dashboard.

## Configuring Ingress with HAProxy

We will use HA Proxy for Ingress because it supports "leastconn" routing (routwe to the instance with the least number of outstanding connections) which is what we want for zqd. 

HAProxy controller wil be installed in it's own namespace. To set this up, we started by pulling this github repo for HAProxy kubernetes ingress:
```
git clone https://github.com/haproxytech/kubernetes-ingress.git
cd kubernetes-ingress/deploy
ls
```
There is an example yaml file that is very clise to what we need. Edit `haproxy-ingress.yaml` to make a single line change, change spec: type: from NodePort to LoadBalancer. Then,
```
kubectl apply -f haproxy-ingress.yaml
```
To verify that the controller is running, look at the pods and services:
```
kubectl get ns
kubectl get pod -n default
kubectl get pod -n haproxy-controller
kubectl get service -n haproxy-controller
```

To deploy zqd with ingress, use folowing helm command. You must set ZQD_ELB to the host for you the ELB created by the HAProxy controller, e.g. a6912345127653265324d48-398735649.us-east-2.elb.amazonaws.com (not a real example.)
```
helm install zqd charts/zqd \
  --set AWSRegion="us-east-2" \
  --set image.repository="$ZQD_ECR_HOST/" \
  --set image.tag="zqd:$ECR_VERSION" \
  --set useCredSecret=false \
  --set datauri=$ZQD_DATA_URI \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=$ZQD_ELB \
  --set ingress.hosts[0].paths={"/"}
```

Test the ingress with,
```
curl $ZQD_ELB
```
And you should get a 404 form the default endpoint. Test the zqd endpoint with zapi:
```
zapi -h $ZQD_ELB ls
```
Should return a list of spaces from zapi.

## Setting up GitOps with ArgoCD

This step is optional (obviously) and is included so developers will know how Brim configures CI/CD on K8s for the zq project.

We use the setup instructions at:
https://argoproj.github.io/argo-cd/

At step 3, we chose Service Type Load Balancer.
