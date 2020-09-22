# Deploying zqd in Kubernetes clusters

**zqd** can be run remotely in a Kubernetes (K8s) cluster. We describe multiple approaches to this.

In order to develop for Kubernetes without paying for a cloud provider, we describe how to set up a local Kind cluster (Kubenertes in Docker).

[Setting up a Kind cluster for zqd](kind-setup.md)

Once the Kind cluster is set up, there are Makefile rules for building a Docker image, and installing the zqd service with Helm.

[How to deploy zqd on Kind](dev-on-kind.md)

If you have access to an AWS account, we describe how to set up an EKS cluster for zqd development and testing.

[Setting up an EKS cluster for zqd](eks-setup.md)

When you have an EKS cluster set up, either using the link above, or using a previously existing cluster, you can use the same Makefile rules for dev deploymemts on zqd into EKS.

[How to deploy zqd on EKS](dev-on-eks.md)

The Brim AWS account is used to automatically deploy and test the master branch of zq in a test EKS cluster. If you fork the repo, you may want to do something similar. We describe our procedures and Github actions for automatc build and test.

(This is work in progress and is incomplete.)

[Github actions for deploying and testing zqd](deploy-with-github-actions.md)

In the process of working all this stuff out, we took a lot of notes on the trouble-shooting steps we sometimes needed. Feel free to dig around for info in here:

[Trouble shooting](troubleshooting.md)

