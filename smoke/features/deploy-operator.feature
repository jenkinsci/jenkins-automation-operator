Feature: Deploy Jenkins Operator on operator hub

  As a user of Jenkins Operator
      I want to deploy as jenkins instance & build by applications

  Background:
    Given Project [TEST_NAMESPACE] is used

  Scenario: Deploy Jenkins Operator
    Given We have a openshift cluster
    When the resources are created using the crd
    Then We create template from yaml
    And Apply template with oc new-app
    Then Check for pod creation and state