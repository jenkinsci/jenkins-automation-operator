# Installing Jenkins Operator

This guide explains how to install jenkins operator.


## Requirements

To run Jenkins Operator, you will need:
  * access to a Kubernetes cluster version 1.11+
  * kubectl version 1.11+

## Install Jenkins Custom Resource Definition

Install Jenkins Custom Resource Definition:

    ```
    kubectl apply -f https://raw.githubusercontent.com/jenkinsci/Kubernetes-operator/master/deploy/crds/jenkins_v1alpha2_jenkins_crd.yaml
    ```

## Deploy Jenkins Operator

There are two ways to deploy the Jenkins Operator:

### Applying the Yaml manifest

    ```
    kubectl apply -f https://raw.githubusercontent.com/jenkinsci/Kubernetes-operator/master/deploy/all-in-one-v1alpha2.yaml
    ```

### Using the helm chart

    ```
    helm repo add jenkins https://raw.githubusercontent.com/jenkinsci/Kubernetes-operator/master/chart
    helm install jenkins/jenkins-operator
    ```

Now Jenkins Operator should be up and running in the default namespace.
