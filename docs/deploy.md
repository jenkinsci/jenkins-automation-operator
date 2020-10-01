# Deploying Jenkins

This guide explains how to deploy Jenkins once Jenkins Operator is up and running.

## Create The Jenkins CR

Create a new Jenkins CR or use an existing example under deploy/crds/Jenkins_v1alpha2_Jenkins_cr.yaml.
For openshift you can use deploy/crds/openshift_Jenkins_v1alpha2_Jenkins_cr.yaml.

## Deploy a Jenkins to Kubernetes
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/Jenkinsci/Kubernetes-operator/master/deploy/crds/Jenkins_v1alpha2_Jenkins_cr.yaml
    ```

## Deploy a Jenkins to openshift
    ```bash
    kubectl apply -f https://raw.githubusercontent.com/Jenkinsci/Kubernetes-operator/master/deploy/crds/openshift_Jenkins_v1alpha2_Jenkins_cr.yaml
    ```

## Check the deployment

Watch the jekins pod

    ```bash
    kubectl get pods -w
    ```

Get the Jenkins credentials:
    ```bash
    kubectl get secret Jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.user}' | base64 -d
    kubectl get secret Jenkins-operator-credentials-<cr_name> -o 'jsonpath={.data.password}' | base64 -d
    ```

Connect to the Jenkins instance (minikube):
    ```bash
    minikube service Jenkins-operator-http-<cr_name> --url
    ```

Connect to the Jenkins instance (Kubernetes):
     ```bash
    kubectl port-forward Jenkins-<cr_name> 8080:8080
    ```
Then open browser with address http://localhost:8080.