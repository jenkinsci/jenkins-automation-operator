Feature: Deploy Jenkins Operator on operator hub using catalog source and new operator bundle format

  As a user of Jenkins Operator
      I want to test if jenkins operator is deployed on operator hub properly
      If we need to install an operator manually using the cli 
        - ensure your catalog source is installed
        - create an OperatorGroup
        - create the Subscription object

  Background:
    Given Project [TEST_NAMESPACE] is used

  Scenario: Deploy Jenkins Operator on operator hub
    Given we have a openshift cluster
    When we create the catalog source using catalog-source.yaml
    Then We create operator group using operator-group.yaml
    And We create subscription using subscriptions.yaml
    Then we check for the csv and csv version
    And we check for the operator group
    And we check for the subscription
    Then we check for the operator pod