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


## Delete the local cluster
When you no longer need the local cluster, you can remove it with:
```
kind delete cluster --name zq-local
```

