# Deploying zqd on Kind

To development K8s deployment and testing, we use Tilt.

https://tilt.dev 

The zq root directory has a Tiltfile that can deploy to Kind. We describe how to use it. Later in this doc, we explain the "manual" steps for deployment if you are not using Tilt, since the manual steps are the way we built the Tiltfile to begin with.

## Deploying with Tilt

```
tilt up
```

## Individual build steps seperate from the Tiltfile

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

We do not need Linkerd for the typical use case of service meshes, which is to route intra-service messages, often gRPC, within a cluster. Instead, we leverage the side-car router as a conveninet way to monitor inbound and outbound traffic from zqd. Conveniently, the Linkerd install also includes Prometheus and Grafana to support Linkerd's dashboard. We will configure these Prometheus and Grafana instances to also monitor zqd's Prometheus metrics.

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
