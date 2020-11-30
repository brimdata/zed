# Using zqd recruiter

Here is a pattern for deploying zqd recruiter in a cluster, and using it as a service that supplied zqd worker instances to perform distributed query execution.

zqd recruiter, workers, and root can be deployed with the zqd helm chart with the following Makefile targets.

Install the recruiter and let it come up before installing the workers.
```
make helm-install-recruiter
```

```
make helm-install-root
make helm-install-workers
```

This will deploy zqd recruiter as a service with a replication count of 1. With its current design, zqd recruiter should not have more than one instance deployed in a cluster. The zqd recruiter is reliable because it can quickly recover state after an expected or unexpected restart.

In a K8s cluster, the zqd worker instances that register with the recruiter should be started with the `-personality=worker` command line flag, and with these environment variables:

```
ZQD_REGISTER=<the host:port to be used for registration by a zqd worker process>
ZQD_HOST=<host ip of the zqd worker pod that is registering>
ZQD_PORT=<port for zqd worker service in the pod>
ZQD_NODE_NAME=<name of the K8s node within the cluster on which the worker pod is deployed>
```

The zqd helm template includes deployment.yaml that provides these env vars.

The zqd recruiter process should be deployed with the following env vars:
```
ZQD_RECRUIT=<the host:port to be used for recruit by a zqd root process>
```

The following environment variables are only used for testing a zqd root instance and should not be used within a K8s deployment:

```
ZQD_TEST_WORKERS=<comma seperated list of host:port for zqd workers being tested>
```
## K8s deployment.yaml

The information we need to send the recruiter is obtained through environment
variables that can be set within the K8s deployment. See this doc:
https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/

## Testing the deployment

```
kubectl port-forward svc/recruiter-zqd 8020:9867 &
curl http://localhost:8020/workers/stats
curl http://localhost:8020/workers/listfree
```




