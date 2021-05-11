## Operator Bundle for OpenShift Jenkins Operator

Build the Bundle image ( operator + OLM manifests )

```
make build
```

[![Container Image Repository on Quay](https://quay.io/repository/redhat-developer/openshift-jenkins-operator-bundle/status "Container Image Repository on Quay")](https://quay.io/repository/redhat-developer/openshift-jenkins-operator-bundle)

## Test Operator Bundle install for Openshift Jenkins operator

Feature 1: Deploy Jenkins Operator on operator hub using catalog source and new operator bundle format
``` 
As a user of Jenkins Operator
      I want to test if jenkins operator is deployed on operator hub properly
      If we need to install an operator manually using the cli 
            - ensure your catalog source is installed
            - create an OperatorGroup
            - create the Subscription object

Scenario: Deploy Jenkins Operator on operator hub
      Given we have a openshift cluster
      When we create the catalog source using catalog-source.yaml
      Then We create operator group using operator-group.yaml
      And We create subscription using subscriptions.yaml
      Then we check for the csv and csv version
      And we check for the operator group
      And we check for the subscription
      Then we check for the operator pod

```
Feature 2: Install jenkins operator
```
As a user of Jenkins Operator
      I want to install a jenkins instance & trigger my jobs 
      
Scenario: Install jenkins instance
      Given Jenkins operator is installed
      When we create the jenkins instance using jenkins.yaml
      Then We check for the jenkins-simple pod status
      And We check for the route

Scenario: Deploy sample application on openshift
      Given The jenkins pod is up and runnning
      When The user enters new-app command with sample-pipeline
      Then Trigger the build using oc start-build
      Then nodejs-mongodb-example pod must come up
      And route nodejs-mongodb-example must be created and be accessible
```
Feature 3: Jenkins with backup enabled
```
As a user of Jenkins Operator
      I want to have a jenkins instance with backup enabled
      
Scenario: Create jenkins with backup
    Given All containers in the jenkins pod are running
    When we check for the default backupconfig
    Then We create backup object using backup.yaml
    And We check for the backup object named example
    Then We rsh into the backup container and check for the jenkins-backups folder contents
```
Feature 4: Testing jenkins agent maven image
```
As a user of Jenkins Operator
    I want to deploy JavaEE application on OpenShift
        

Background:
      Given Project [TEST_NAMESPACE] is used

Scenario: Deploy JavaEE application on OpenShift
      Given The jenkins pod is up and runnning
      When The user create objects from the sample maven template by processing the template and piping the output to oc create
      And verify imagestream.image.openshift.io/openshift-jee-sample & imagestream.image.openshift.io/wildfly exist
      And verify buildconfig.build.openshift.io/openshift-jee-sample & buildconfig.build.openshift.io/openshift-jee-sample-docker exist
      And verify deploymentconfig.apps.openshift.io/openshift-jee-sample is created
      And verify service/openshift-jee-sample is created
      And verify route.route.openshift.io/openshift-jee-sample is created
      Then Trigger the build using oc start-build openshift-jee-sample
      Then verify the build status of openshift-jee-sample-docker build is Complete
      And verify the build status of openshift-jee-sample build is Complete
      And verify the JaveEE application is accessible via route openshift-jee-sample

```
## Run the Smoke Test

```
- export KUBECONFIG=<path/to/kubeconfig>
- make smoke
```
[![Container Image Repository on Quay](https://quay.io/repository/redhat-developer/openshift-jenkins-operator-index/status "Operator Index Image Repository on Quay")](https://quay.io/repository/redhat-developer/openshift-jenkins-operator-index)

