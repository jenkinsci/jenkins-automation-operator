Feature: Install jenkins operator

  As a user of Jenkins Operator
      I want to install a jenkins instance & trigger my jobs 
        

  Background:
    Given Project [TEST_NAMESPACE] is used

  Scenario: Install jenkins instance
    Given Jenkins operator is installed
    When we create the jenkins instance using jenkins.yaml
    Then We check for the jenkins-example pod status
    And We check for the route

  Scenario: Deploy sample application on openshift
      Given The jenkins pod is up and runnning
      When The user enters new-app command with sample-pipeline
      Then Trigger the build using oc start-build
      Then nodejs-mongodb-example pod must come up
      And route nodejs-mongodb-example must be created and be accessible