---
title: "Building Jenkins Images"
linkTitle: "Building Jenkins Images"
weight: 20
date: 2020-06-30
description: >
  Building your own Jenkins Images
---

**Jenkins Operator** provides a convenient way to build your own Jenkins images using a the custom resource **JenkinsImage**.
The JenkinsImage controller allows you to build any jenkins:lts compatible image by specifying a list of plugins 
and their versions; as well as pushing this image to the registry of your choice.

**JenkinsImages** allows to always keep snapshot of the Jenkins bits (base and plugins) that you are running. 
This allows to speed-up redeployment, permit to have strictly identical Jenkins instances and know what you 
are running at any time.

## How it works
The build mechanism relies on **kaniko** which provides a convenient way to build container 
images without relying on the docker socket or any daemon based container engine. The **Jenkins Image Controller** 
monitors the creation of a **JenkinsImage** object and it creates a configMap containing 
the generated Dockerfile used by **kaniko** as an input.
The build is ran into a Pod named <cr.Name-builder> and once the build succeeds, it pushes the resulting image 
to the specified registry at the specified name with the specified tag.

TODO: The **Jenkins** CR will be updated to allow passing a reference to a JenkinsImage.

## Example of a JenkinsImage

This example shows a simple JenkinsImage containing a few plugins. It requires that a secret 
named `my-quay-credentials` to exist and to be in the config.json format of docker:
```json
{
  "auths": {
    "quay.io": {
      "auth": "xxxxxxx==",
      "email": ""
    }
  }
}
```
The corresponding secret can be created using:
```shell
kubectl create secret generic  my-quay-credentials  --from-file=config.json
```

And the JenkinsImage would be: 

```yaml
apiVersion: jenkins.io/v1alpha2
kind: JenkinsImage
metadata:
  name: simple-jenkinsimage
spec:
  from:
    name: jenkins/jenkins
    tag: lts
  plugins:
  - name: kubernetes
    version: "1.15.7"
  - name: workflow-job 
    version: "2.32"
  - name: workflow-aggregator
    version: "2.6"
  - name: git
    version: "3.10.0"
  - name: job-dsl
    version: "1.74"
  - name: configuration-as-code
    version: "1.19"
  - name: kubernetes-credentials-provider
    version: "0.12.1"
  to:
    registry: quay.io/akram
    name: jenkins-for-jim
    tag: latest
    secret: my-quay-credentials
```


