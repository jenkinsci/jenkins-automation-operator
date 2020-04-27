---
title: "Developer Guide"
linkTitle: "Developer Guide"
weight: 60
date: 2019-08-05
description: >
  Jenkins Operator for developers
---

{{% pageinfo %}}
This document explains how to setup your development environment.
{{% /pageinfo %}}

## Prerequisites

- [operator_sdk][operator_sdk] version v0.15.1
- [git][git_tool]
- [go][go_tool] version v1.13+
- [goimports, golint, checkmake and staticcheck][install_dev_tools]
- [minikube][minikube] version v1.1.0+ (preferred Hypervisor - [virtualbox][virtualbox])
- [docker][docker_tool] version 17.03+

## Clone repository and download dependencies

```bash
git clone git@github.com:jenkinsci/kubernetes-operator.git
cd kubernetes-operator
make go-dependencies
```

## Build and run with a minikube

Build and run **Jenkins Operator** locally:

```bash
make build minikube-run

INFO[0000] Running deepcopy code-generation for Custom Resource group versions: [jenkins:[v1alpha2], ] 
INFO[0005] Code-generation complete.                    
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:51	Version: v0.4.0
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:52	Git commit: 4ffc58e-dirty
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:53	Go Version: go1.13.1
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:54	Go OS/Arch: linux/amd64
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:55	operator-sdk Version: v0.15.1
2020-04-27T09:52:26.520+0200	INFO	controller-jenkins	manager/main.go:80	Watch namespace: default
2020-04-27T09:52:26.527+0200	INFO	leader	leader/leader.go:46	Trying to become the leader.
2020-04-27T09:52:26.527+0200	INFO	leader	leader/leader.go:51	Skipping leader election; not running in a cluster.
2020-04-27T09:52:26.887+0200	INFO	controller-runtime.metrics	metrics/listener.go:40	metrics server is starting to listen	{"addr": "0.0.0.0:8383"}
2020-04-27T09:52:26.887+0200	INFO	controller-jenkins	manager/main.go:105	Registering Components.
2020-04-27T09:52:26.897+0200	WARN	controller-jenkins	manager/main.go:138	Could not generate and serve custom resource metrics	{"error": "namespace not found for current environment"}
2020-04-27T09:52:27.250+0200	INFO	metrics	metrics/metrics.go:55	Skipping metrics Service creation; not running in a cluster.
2020-04-27T09:52:27.601+0200	WARN	controller-jenkins	manager/main.go:157	Could not create ServiceMonitor object	{"error": "no ServiceMonitor registered with the API"}
2020-04-27T09:52:27.601+0200	WARN	controller-jenkins	manager/main.go:161	Install prometheus-operator in your cluster to create ServiceMonitor objects	{"error": "no ServiceMonitor registered with the API"}
2020-04-27T09:52:27.601+0200	INFO	controller-jenkins	manager/main.go:165	Starting the Cmd.
2020-04-27T09:52:27.601+0200	INFO	controller-runtime.manager	manager/internal.go:356	starting metrics server	{"path": "/metrics"}
2020-04-27T09:52:27.601+0200	INFO	controller-runtime.controller	controller/controller.go:164	Starting EventSource	{"controller": "jenkins-controller", "source": "kind source: jenkins.io/v1alpha2, Kind=Jenkins"}
2020-04-27T09:52:27.702+0200	INFO	controller-runtime.controller	controller/controller.go:164	Starting EventSource	{"controller": "jenkins-controller", "source": "kind source: core/v1, Kind=Pod"}
2020-04-27T09:52:27.803+0200	INFO	controller-runtime.controller	controller/controller.go:164	Starting EventSource	{"controller": "jenkins-controller", "source": "kind source: core/v1, Kind=Secret"}
2020-04-27T09:52:27.903+0200	INFO	controller-runtime.controller	controller/controller.go:164	Starting EventSource	{"controller": "jenkins-controller", "source": "kind source: core/v1, Kind=Secret"}
2020-04-27T09:52:27.903+0200	INFO	controller-runtime.controller	controller/controller.go:164	Starting EventSource	{"controller": "jenkins-controller", "source": "kind source: core/v1, Kind=ConfigMap"}
2020-04-27T09:52:28.005+0200	INFO	controller-runtime.controller	controller/controller.go:171	Starting Controller	{"controller": "jenkins-controller"}
2020-04-27T09:52:28.005+0200	INFO	controller-runtime.controller	controller/controller.go:190	Starting workers	{"controller": "jenkins-controller", "worker count": 1}
```

```bash
kubectl apply -f deploy/crds/jenkins_v1alpha2_jenkins_cr.yaml

2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:404	Setting default Jenkins container command	{"cr": "example"}
2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:409	Setting default Jenkins container JAVA_OPTS environment variable	{"cr": "example"}
2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:417	Setting default operator plugins	{"cr": "example"}
2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:436	Setting default Jenkins master service	{"cr": "example"}
2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:449	Setting default Jenkins slave service	{"cr": "example"}
2020-04-27T09:56:40.153+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:479	Setting default Jenkins API settings	{"cr": "example"}
2020-04-27T09:56:40.158+0200	INFO	controller-jenkins	jenkins/handler.go:89	*v1alpha2.Jenkins/example has been updated	{"cr": "example"}
2020-04-27T09:56:40.562+0200	INFO	controller-jenkins	base/pod.go:161	Creating a new Jenkins Master Pod default/jenkins-example	{"cr": "example"}
2020-04-27T09:56:40.575+0200	INFO	controller-jenkins	base/reconcile.go:528	The Admission controller has changed the Jenkins master pod spec.securityContext, changing the Jenkinc CR spec.master.securityContext to '&PodSecurityContext{SELinuxOptions:nil,RunAsUser:nil,RunAsNonRoot:nil,SupplementalGroups:[],FSGroup:nil,RunAsGroup:nil,Sysctls:[]Sysctl{},WindowsOptions:nil,}'	{"cr": "example"}
2020-04-27T09:56:40.584+0200	INFO	controller-jenkins	jenkins/handler.go:89	*v1alpha2.Jenkins/example has been updated	{"cr": "example"}
2020-04-27T09:59:40.409+0200	INFO	controller-jenkins	base/reconcile.go:466	Generating Jenkins API token for operator	{"cr": "example"}
2020-04-27T09:59:40.410+0200	WARN	controller-jenkins	jenkins/jenkins_controller.go:171	Reconcile loop failed: couldn't init Jenkins API client: Get http://192.168.99.100:32380/api/json: dial tcp 192.168.99.100:32380: connect: connection refused	{"cr": "example"}
2020-04-27T09:59:40.455+0200	INFO	controller-jenkins	base/reconcile.go:466	Generating Jenkins API token for operator	{"cr": "example"}
2020-04-27T09:59:41.415+0200	INFO	controller-jenkins	groovy/groovy.go:145	base-groovy ConfigMap 'jenkins-operator-base-configuration-example' name '1-basic-settings.groovy' running groovy script	{"cr": "example"}
...
2020-04-27T09:59:49.030+0200	INFO	controller-jenkins	groovy/groovy.go:145	base-groovy ConfigMap 'jenkins-operator-base-configuration-example' name '8-disable-job-dsl-script-approval.groovy' running groovy script	{"cr": "example"}

2020-04-27T09:59:49.257+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:289	Base configuration phase is complete, took 3m9s	{"cr": "example"}
2020-04-27T09:59:51.165+0200	INFO	controller-jenkins	seedjobs/seedjobs.go:232	Waiting for Seed Job Agent `seed-job-agent`...	{"cr": "example"}
...
2020-04-27T10:00:03.886+0200	INFO	controller-jenkins	seedjobs/seedjobs.go:232	Waiting for Seed Job Agent `seed-job-agent`...	{"cr": "example"}
2020-04-27T10:00:06.140+0200	INFO	controller-jenkins	jenkins/jenkins_controller.go:338	User configuration phase is complete, took 3m26s	{"cr": "example"}
```

Two log lines says that Jenkins Operator works correctly:
 
* `Base configuration phase is complete` - ensures manifests, Jenkins pod, Jenkins configuration and Jenkins API token  
* `User configuration phase is complete` - ensures Jenkins restore, backup and seed jobs along with user configuration 

> Details about base and user phase can be found [here](https://jenkinsci.github.io/kubernetes-operator/docs/how-it-works/architecture-and-design/).


```bash
kubectl get jenkins -o yaml

apiVersion: v1
items:
- apiVersion: jenkins.io/v1alpha2
  kind: Jenkins
  metadata:
    ...
  spec:
    backup:
      action: {}
      containerName: ""
      interval: 0
      makeBackupBeforePodDeletion: false
    configurationAsCode:
      configurations: null
      secret:
        name: ""
    groovyScripts:
      configurations: null
      secret:
        name: ""
    jenkinsAPISettings:
      authorizationStrategy: createUser
    master:
      basePlugins:
        ...
      - command:
        - bash
        - -c
        - /var/jenkins/scripts/init.sh && exec /sbin/tini -s -- /usr/local/bin/jenkins.sh
        env:
        - name: JAVA_OPTS
          value: -XX:+UnlockExperimentalVMOptions -XX:+UseCGroupMemoryLimitForHeap
            -XX:MaxRAMFraction=1 -Djenkins.install.runSetupWizard=false -Djava.awt.headless=true
        image: jenkins/jenkins:lts
        imagePullPolicy: Always
        livenessProbe:
         ...
        name: jenkins-master
        readinessProbe:
         ...
        resources:
          limits:
            cpu: 1500m
            memory: 3Gi
          requests:
            cpu: "1"
            memory: 500Mi
      disableCSRFProtection: false
      securityContext: {}
    restore:
      action: {}
      containerName: ""
    seedJobs:
    - additionalClasspath: ""
      bitbucketPushTrigger: false
      buildPeriodically: ""
      description: Jenkins Operator repository
      failOnMissingPlugin: false
      githubPushTrigger: false
      id: jenkins-operator
      ignoreMissingFiles: false
      pollSCM: ""
      repositoryBranch: master
      repositoryUrl: https://github.com/jenkinsci/kubernetes-operator.git
      targets: cicd/jobs/*.jenkins
      unstableOnDeprecation: false
    service:
      port: 8080
      type: NodePort
    serviceAccount: {}
    slaveService:
      port: 50000
      type: ClusterIP
  status:
    appliedGroovyScripts:
    - configurationType: base-groovy
      hash: 2ownqpRyBjQYmzTRttUx7axok3CKe2E45frI5iRwH0w=
      name: 1-basic-settings.groovy
      source: jenkins-operator-base-configuration-example
        ...
    baseConfigurationCompletedTime: "2020-04-27T07:59:49Z"
    createdSeedJobs:
    - jenkins-operator
    operatorVersion: v0.4.0
    provisionStartTime: "2020-04-27T07:56:40Z"
    userAndPasswordHash: kAeBnhHKU3LZuw+uo9oHILB59kAFSGDUbHwCSDgtMnE=
    userConfigurationCompletedTime: "2020-04-27T08:00:06Z"
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

```bash
kubectl get po

NAME                                      READY   STATUS    RESTARTS   AGE
jenkins-example                           1/1     Running   0          15m
seed-job-agent-example-56569459c9-l69qf   1/1     Running   0          12m

```

### Debug Jenkins Operator

```bash
make build minikube-run OPERATOR_EXTRA_ARGS="--debug"
```

## Build and run with a remote Kubernetes cluster

You can also run the controller locally and make it listen to a remote Kubernetes server.

```bash
make run NAMESPACE=default KUBECTL_CONTEXT=remote-k8s EXTRA_ARGS='--kubeconfig ~/.kube/config'
```

Once **Jenkins Operator** are up and running, apply Jenkins custom resource:

```bash
kubectl --context remote-k8s --namespace default apply -f deploy/crds/jenkins_v1alpha2_jenkins_cr.yaml
kubectl --context remote-k8s --namespace default get jenkins -o yaml
kubectl --context remote-k8s --namespace default get po
```

## Testing

Run unit tests:

```bash
make test
```

### Running E2E tests

Run e2e tests with minikube:

```bash
make minikube-start
eval $(minikube docker-env)
make e2e
```

Run the specific e2e test:

```bash
make build e2e E2E_TEST_SELECTOR='^TestConfiguration$'
```

## Tips & Tricks

### Building docker image on minikube (for e2e tests)

To be able to work with the docker daemon on `minikube` machine run the following command before building an image:

```bash
eval $(minikube docker-env)
```

### When `pkg/apis/jenkinsio/*/jenkins_types.go` has changed

Run:

```bash
make deepcopy-gen
```

### Getting the Jenkins URL and basic credentials

```bash
minikube service jenkins-operator-http-<cr_name> --url
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
kubectl get secret jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
```

[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[operator_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[minikube]:https://kubernetes.io/docs/tasks/tools/install-minikube/
[virtualbox]:https://www.virtualbox.org/wiki/Downloads
[install_dev_tools]:https://jenkinsci.github.io/kubernetes-operator/docs/developer-guide/tools/

## Self-learning

* [Tutorial: Deep Dive into the Operator Framework for... Melvin Hillsman, Michael Hrivnak, & Matt Dorn
](https://www.youtube.com/watch?v=8_DaCcRMp5I)

* [Operator Framework Training By OpenShift](https://www.katacoda.com/openshift/courses/operatorframework)
