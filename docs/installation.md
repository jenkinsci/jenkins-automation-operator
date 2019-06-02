# Installation

This document describes installation procedure for **jenkins-operator**.
All container images can be found at [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator)

## Requirements
 
To run **jenkins-operator**, you will need:
- running Kubernetes cluster
- kubectl

## Configure Custom Resource Definition 

Install Jenkins Custom Resource Definition:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/crds/jenkinsio_v1alpha1_jenkins_crd.yaml
```

## Deploy jenkins-operator

Apply Service Account and RBAC roles:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/service_account.yaml
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/role.yaml
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/role_binding.yaml
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/operator.yaml
```

Watch **jenkins-operator** instance being created:

```bash
kubectl get pods -w
```

Now **jenkins-operator** should be up and running in `default` namespace.



