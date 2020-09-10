# Deploying zqd on Kind

To development K8s deployment and testing, we use Tilt.

https://tilt.dev 

The zq root directory has a Tiltfile that can deploy to Kind. We describe how to use it. Later in this doc, we explain the "manual" steps for deployment if you are not using Tilt, since the manual steps are the way we built the Tiltfile to begin with.

## Deploying with Tilt

```
tilt up
```

## Creating test data in S3 using zar
Before using zapi to access the running zqd, we use `zar import` in another console to copy sample data from our zq repo into s3. You must have an AWS account and the AWS CLI on your desktop machine to do this. Change the directory name to match your s3 bucket.
```
zar import -R s3://brim-scratch/mark/sample-http-zng zq-sample-data/zng/http.zng.gz
```
This creates zng files in an s3 directory called `sample-http-zng` that we will use from zapi. To check what zar created:
```
aws s3 ls brim-scratch/mark --recursive
```

## Port forwarding for local testing
To test locally, run this script to forward the Kind/K8s ports to local ports:
```
./k8s/ports.sh
```

## Testing the deployed zqd with zapi and Brim
Now use zapi to create a Brim "space":
```
zapi new -k archivestore -d s3://brim-scratch/mark/sample-http-zng http-space
```
And try some zapi queries:
```
zapi -s http-space get "head 1"
zapi -s http-space get "tail 1"
```

We can also query http-space with Brim, since it will connect to the same port-forward for zqd.

## Individual build steps seperate from the Tiltfile

This explains each of the parts of the Tiltfile -- you do not need to run these steps if `tilt up` is doing everything you need.

### Build a zqd image with Docker
There is a Dockerfile in the root directory. This Dockerfile does a 2-stage build to produce a relatively small image containing the zqd binary. It is structured to cache intermediate results so it runs faster on subsequent builds.

The Makefile has a target `make docker` that creates a docker image with the correct version number passed in as LDFLAGS. `make docker` is the preferred way to build a docker image.

The `make docker` target also copies the image into a Docker registry that is accessed by your single-node Kubernetes cluster for Kind.


### AWS credentials

To access AWS S3 from zqd running as a Kubernetes pod, you must have AWS credentials available to the code running in the pod. We use K8s secrets to provide the credentials to the deployment as env vars. The secrets are referenced in the Helm template deployment.yaml. The shell script `aws-credentials.sh` reads your credentials using `aws configure` and creates a K8s secret in the Kind cluster. (We will do something different for AWS EKS, because we can use IAM when the cluster is deployed in AWS.)

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

