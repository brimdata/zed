# Using zqd recruiter

Here is a pattern for deploying zqd recruiter in a cluster, and using it as a service that supplied zqd worker instances to perform distributed query execution.

zqd recruiter can be deployed with the zqd helm chart with the following options:

```
helm ... (TBD)
```

This will deploy zqd recruiter as a service with a replication count of 1. With its current design, zqd recruiter should not have more than one instance deployed in a cluster. The zqd recruiter is reliable because it can quickly recover state after an expected or unexpected restart.

In a K8s cluster, the zqd worker instances that register with the recruiter should be started with the `-personality=worker` command line flag, and with these environment variables:

```
ZQD_RECRUITER=<the host:port of the zqd recruiter K8s service within the cluster>
ZQD_ADDR=<host:port of the zqd worker pod that is registering>
ZQD_NODE_NAME=<name of the K8s node within the cluster on which the pod is deployed>
```

The zqd helm template includes deployment.yaml that provides these env vars.

The following environment variables are only used for testing a zqd root instance and should not be used within a K8s deployment:

```
ZQD_TEST_WORKERS=<comma seperated list of host:port for zqd workers being tested>
```
## K8s deployment.yaml

The information we need to send the recruiter is obtained through environment
variables that can be set within the K8s deployment. See this doc:
https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/

