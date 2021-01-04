# Deploying zqd on Kind

This assumes you have already followed the instructions in [Setting up a Kind cluster for zqd](kind-setup.md).

## Build and upload docker image
To build and push the zqd image to the local Docker repo that is deployed on Kind, use:
```
make docker-push-local
```

## Install Postgres with Helm

Because the helm recipe for postgres uses a persistent volume claim to persist
the database between installs, we must create a kubernetes secret with postgres
passwords that will also persist between installs. Run this script to create
a secret with randomly generated passwords for the postgres admin and zqd user
accounts:

```
./k8s/postgres-secret.sh
```

The postgres database can now be installed via helm:

```
make helm-install-postgres
```

## Install with Helm
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

## Port forwarding for local testing
To test locally, run this script to forward the Kind/K8s ports to local ports:
```
./k8s/zqd-port.sh
```

## Testing the deployed zqd with zapi and Brim
Now use zapi to create a Brim "space":
```
zapi new -k archivestore http-space
zapi -s http-space post s3://zq-sample-data/zng/http.zng.gz
```
And try some zapi queries:
```
zapi -s http-space get "head 1"
zapi -s http-space get "tail 1"
```

We can also query http-space with Brim, since it will connect to the same port-forward for zqd.

## Appendix: Individual build steps

This give more detail on the Makefile rules.

### Build a zqd image with Docker
There is a Dockerfile in the root directory. This Dockerfile does a 2-stage build to produce a relatively small image containing the zqd binary. It is structured to cache intermediate results so it runs faster on subsequent builds.

The Makefile has a target `make docker` that creates a docker image with the correct version number passed in as LDFLAGS. `make docker` is the preferred way to build a docker image.

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

