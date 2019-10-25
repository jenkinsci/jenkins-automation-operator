---
title: "Installation"
linkTitle: "Installation"
weight: 1
date: 2019-08-05
description: >
  How to install Jenkins Operator
---

{{% pageinfo %}}
This document describes installation procedure for **Jenkins Operator**.
All container images can be found at [virtuslab/jenkins-operator](https://hub.docker.com/r/virtuslab/jenkins-operator)
{{% /pageinfo %}}

## Requirements
 
To run **Jenkins Operator**, you will need:

- running Kubernetes cluster version 1.11+

- kubectl version 1.11+

## Configure Custom Resource Definition 

Install Jenkins Custom Resource Definition:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/crds/jenkins_v1alpha2_jenkins_crd.yaml
```

## Deploy Jenkins Operator

Apply Service Account and RBAC roles:

```bash
kubectl apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/all-in-one-v1alpha2.yaml
```

Watch **Jenkins Operator** instance being created:

```bash
kubectl get pods -w
```

Now **Jenkins Operator** should be up and running in `default` namespace.
