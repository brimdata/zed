# Using zqd recruiter
A simple pattern for deploying and testing the recruiter is in:
```
./k8s/demo.sh
```
Which uses helm and Makefile rules to deploy a recruiter, zqd workers, and a zqd root.
The Makefile rules and the helm templates illustrate the deployment patterns required.

This will deploy zqd recruiter as a service with a replication count of 1. With its current design, zqd recruiter should not have more than one instance deployed in a cluster. The zqd recruiter is reliable because it can quickly recover state after an expected or unexpected restart.

The following environment variable is only used for ZTest testing of a zqd root instance and should not be used within a K8s deployment:
```
ZQD_TEST_WORKERS=<comma seperated list of host:port for zqd workers being tested>
```
## K8s deployment.yaml
The information we need to send the recruiter is obtained through environment
variables that can be set within the K8s deployment. See this doc:
https://kubernetes.io/docs/tasks/inject-data-application/environment-variable-expose-pod-information/

## Useful commands for inspecting the state of the recruiter:
```
kubectl port-forward svc/recruiter-zqd 8020:9867 &
curl http://localhost:8020/workers/stats
curl http://localhost:8020/workers/listfree
```
