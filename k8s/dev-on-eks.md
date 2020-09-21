# Deploying zqd on EKS

These are instructions for deploying oa locally built image for zqd into an EKS cluster, in a nameespace that is specific for your development. The deployment is automated by Tilt, but there some inital setup to insure that you have a namespace and a kubect config context for the deployment.

First connect to the EKS cluster so you have kubectl access. Yu or a collegue should follow the steps here to obtain cluster access:

https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html

Then create a namespace specific to you (your name works well) and create a local config context for the namespace. You must name the local config context "zqtest" because that is what the Tiltfile expects:
```
kubectl create namespace myuser
kubectl config set-context zqtest \
  --namespace=myuser \
  --cluster=zq-test.us-east-1.eksctl.io \
  --user=myuser@zq-test.us-east-1.eksctl.io
kubectl config use-context zqtest
```

Now start Tilt, in the zq home directory where the Tiltfile is, with the following command:
```
tilt up
```
You can monitor the output of Tilt at `http://localhost:10350/`

The Tiltfile is kind os simplified distributed Makefile with useful features to support Docker builds and K8s deployments. It uses a syntax simlar to Python.