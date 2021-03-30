# Troubleshooting

These are assorted notes on trouble-shooting and dev procedures. Most of this text was originally in the README.md.

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

## Building with  Docker directly
The following command builds and labels the container image:
```
DOCKER_BUILDKIT=1 docker build --pull --rm \
  -t "zqd:latest" -t "localhost:5000/zqd:latest" .
```
Notice this adds a tags for loading the image into the local docker registry created by `kind-with-registry.sh.` To load the image into the registry:
```
docker push "localhost:5000/zqd:latest"
```
This copies the image into a Docker registry that is accessed by your single-node Kubernetes cluster for Kind.

## Redeploy with helm
If you make a change to the helm templates, you generally will want to uninstall/reinstall zqd.
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

## Using zqd with zapi
This is a walk-through of a local test before testing on our k8s cluster. This is outside the k8s cluster to get familiar with trouble-shooting.

As a prerequisite, you must have an S3 bucket and directory available for zqd. In the S3 console, or at the command line, create a 'datauri' for use with zqd. This directory will hold metadata for the spaces you will create with zapi.

You will need AWS credentials. On your local machine you can use `aws configure` and then you must set the environment variable, `AWS_SDK_LOAD_CONFIG=true`. Within Kubernetes, we will need to handle AWS credentials with secrets. More on that later.

Here is an example of creating a datauri, using an s3 bucket called `brim-scratch` in a directory called `mark` (change the s3 buckets for your setup.)
```
aws s3 ls # make sure the bucket exists!
zqd listen -data s3://brim-scratch/mark/zqd-meta
```
zqd will stay running in that console, listening at `localhost:9867` by default.
zqd will not create any s3 objects in zqd-meta until we issue a zapi command. Before using zapi, we use `zar import` in another console to copy sample data from our zq repo into s3:
```
zar import -R s3://brim-scratch/mark/sample-http-zng zed-sample-data/zng/http.zng.gz
```
This creates zng files in an s3 directory called `sample-http-zng` that we will use from zapi. To check what zar created:
```
aws s3 ls brim-scratch/mark --recursive
```
Now use zapi to make the zar data available to your running zqd instance. Note that zapi defaults to `localhost:9867` for connecting to zqd.
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space-1
```
The `-d` parameter provides the same s3 dir that we used in the `-R` parameter of the zar command above. This command creates a new space for zqd called `http-space-1`. If we again list the s3 directories with `aws s3 ls brim-scratch/mark --recursive` we will see that zqd has now created a new object under its datauri, a directory name that is a base64 string prefixed with "sp_1g0".

Now that you have a zqd running with a space, and you have made an archive from zar data, you can use zapi commands to query the sample data. Example:
```
zapi -s http-space-1 get "head 1"
zapi -s http-space-1 get "tail 1"
```
The zapi commands in quotes are the same as Brim queries. Notice that "head 1" runs much faster than "tail 1", which has to read more data from s3.

## Deploy zqd into the local cluster with helm
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
## Testing connectivity to zqd
A simple test for zqd is to send a /status request to its http endpoint. This is used for the K8s liveness probe. Substitute the pod id into a port-forward command:

```
kubectl get pod
kubectl port-forward zqd-test-1-66b5f9dc8-8dshh 9867:9867
```
And in another console, use curl:
```
curl -v http://localhost:9867/status
```
You should get an 'ok' response.

## Trying out Kube State Metrics
These instructions are digested from:
https://github.com/kubernetes/kube-state-metrics/blob/master/README.md#setup

You must have a go dev environment set up to follow these instructions.

First clone the repo for kube-state-metrics:
```
git clone https://github.com/kubernetes/kube-state-metrics.git
cd kube-state-metrics
```

To do a "vanilla" install into your local Kind cluster, use the configuration in examples/standard:
```
kubectl apply -f examples/standard/
kubectl get pod -n kube-system
```
The `get pod` will show you that the kube-state-metrics are running alongside the scheduler, et. al., in the kube-system namespace. KSM runs in this namespace because it is a core Kubernetes project.

Prometheus is already running in the linkerd namespace. Now we will configure it to scrape KSM. Start by editing the configmap for the Prometheus intance installed by linkerd:
```
kubectl edit configmap -n linkerd linkerd-prometheus-config
```
At the start of the `scrape_configs:` add the following job. Make sure the indentation exactly matches the jobs below it (yaml is picky.)
```
    - job_name: 'kube-state-metrics'
      static_configs:
      - targets: ['kube-state-metrics.kube-system.svc.cluster.local:8080']
```
After adding the text, it should look something like this in context:
```
    scrape_configs:
    - job_name: 'kube-state-metrics'
      static_configs:
      - targets: ['kube-state-metrics.kube-system.svc.cluster.local:8080']

    - job_name: 'prometheus'
```
Now we need to verify that Prometheus is scraping the KSM metrics. 

## Trouble-shooting Prometheus metrics

We will connect to the expression browser for Prometheus by forwarding the port from the pod.
```
export PROM_POD=$(kubectl -n linkerd get pod -l "linkerd.io/proxy-deployment=linkerd-prometheus" -o jsonpath="{.items[0].metadata.name}")
kubectl -n linkerd port-forward $PROM_POD 9090:9090
```
Now in a web browser, get:
```
http://localhost:9090/graph
```
You will see the Prometheus expression browser.

The container metrics from cAdvisor are interesting to us. In the expression browser, in the drop down next to the execute button, select `container_cpu_usage_seconds_total`. Clicj execute and look at the graphs. We will find the CPU usage for our zqd container in this list.

## AWS IAM users and RBAC

The following is a really good explaination of using K8s RBAC with AWS:

https://www.eksworkshop.com/beginner/090_rbac/intro/

This is a tricky topic and the docs on RBAC can be confusing -- the "tutorial" approach makes the connection between IAM and RBAC more clear.

Later in the same doc, there a good explaination of creating a service account:

https://www.eksworkshop.com/beginner/110_irsa/preparation/

### Create an IAM service account to run the zqd service

If you want to use a service account, the following eksctl command has to be done once for your cluster. Substitute in the name of the EKS cluster you created above.
```
eksctl utils associate-iam-oidc-provider --cluster zqtest --approve
```

Create the service account with:
```
eksctl create iamserviceaccount \
  --attach-policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess \
  --approve --override-existing-serviceaccounts \
  --cluster zqdev2 --namespace zq --name zqd-service-account
```
This creates a CloudFormation stack tp provision your service account. Later you can edit the CF stack directly and reapply it to added more policy ARNs.

The `iamserviceaccount` creates three things: an IAM account, an IAM role for that account that has the attached policies, and the K8s `sa` object. Verify the service account has been created with:
```
kubectl get sa
kubectl get sa zqd-service-account -oyaml
```

## SSH into EKS nodes
To do this yu will have to allow SSH access in the cluster.yaml. You will also need to specify a PEM file in the cluster.yaml.

https://docs.aws.amazon.com/cli/latest/userguide/cli-services-ec2-keypairs.html#creating-a-key-pair
After creating the pem file, extract the public key. If you already have a preferred key pair, just extract the public key.
```
aws ec2 create-key-pair --key-name zqKeyPair --query 'KeyMaterial' --output text > zq-eks-test.pem
ssh-keygen -y -f zq-eks-test.pem > zq-eks-test.pub
```

## Scaling EKS cluster
If you keep the cluster up during dev and testing, you may want to manually scale it down with:
```
eksctl scale nodegroup --cluster test -m 1 -N 1 -M 1 -n standard-workers --region us-east-1
```
This decreases the maximum to 1, which is as low as you can go with an EKS cluster.

## Notes on IAM policies
The zqd service account needs read access to S3. AWS IAM has an ARN for `AmazonS3ReadOnlyAccess`. Here is how you get the ARN:
```
aws iam list-policies --query 'Policies[?PolicyName==`AmazonS3ReadOnlyAccess`].Arn'
```
And the ARN is:
```
arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
```
## S3 Error
The following message is issued by the S3 access layer in AWS, and it is not very informatve. :-)
```
AccessDenied: Access Denied
	status code: 403, request id: 63A062E7523BF781, host id: HMF3X7yVYSRF9JqPnvwQpvggZM/JlW3ZQR2BEQB+Jo0+bpONyeqNxSVXcDxVzj794Zld7+9eA9g=
```

## Using centralized logging
The default logging on K8s cluster just uses in-memory logs for the deployed pods. This is inconvenient for trouble-shooting. There are a number of centralized logging services available. The free verion of Papertrail (https://www.papertrail.com) is an easy place to start. If you create a free Papertrail account, the following instructions work for adding the log output from your pods in the EKS cluster to the Papertrail stream:
https://help.papertrailapp.com/kb/configuration/configuring-centralized-logging-from-kubernetes/

These are the kubectl commands. Substitute your account ID for XXXXX
```
kubectl create secret generic papertrail-destination --from-literal=papertrail-destination=syslog+tls://logsN.papertrailapp.com:XXXXX
kubectl create -f https://help.papertrailapp.com/assets/files/papertrail-logspout-daemonset.yml
```

## Creating a tmpfs on an EC2 instance
This can be useful for isolating IO bottlenecks in long-running operations like zar import.

```
sudo mkdir /mnt/ramdisk
sudo mount -t tmpfs -o size=1024m tmpfs /mnt/ramdisk
# Copy some S3 data to /mnt/ramdisk
time aws s3 cp s3://brim-sampledata/wrccdc/zeek-logs/dns.log.gz /mnt/ramdisk/dns.log.gz
```

## whoami on AWS
```
aws sts get-caller-identity
```

## Setting up an EC2 instance to run zar import

Create an EC2 instance in the same region as the S3 buckets you want to access. SSH into the instance, and:
```
# provide your credentials
aws configure
# make sure you can see the S3 buckets
aws s3 ls
# install golang tools from https://golang.org/dl/
wget https://golang.org/dl/go1.16.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.16.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.bash_profile
source ~/.bash_profile
go version  # make sure go is there!
# install git, clone and install zq
sudo yum install git -y
git clone https://github.com/brimdata/zed
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

# and a 14 GB log -- takes longer...
time ~/go/bin/zar import -R s3://zqd-demo-1/mark/zeek-logs/conn s3://brim-sampledata/wrccdc/zeek-logs/conn.log.gz
```
Assuming you have an EKS cluster set up as described above, with the port-forward in effect,  you can use zapi to create "spaces" for Brim:
```
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/dpd dpd-space
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/dns dns-space
zapi new -k archivestore -d s3://zqd-demo-1/mark/zeek-logs/conn conn-space
```

Now you can run Brim, and it will use the same local:9867 port used by zapi. You will see the three spaces you just created, dpd-space, dns-space, and conn-space.

## Finding everything in a namespace
```
kubectl api-resources --verbs=list --namespaced -o name \
  | xargs -n 1 kubectl get --show-kind --ignore-not-found -n <namespace>
```
