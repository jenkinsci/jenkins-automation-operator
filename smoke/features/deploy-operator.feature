Feature: Deploy Jenkins Operator on operator hub
  Scenario: Deploy Jenkins Operator
    Given We have a openshift cluster
    Then we build the jenkins operator image
    And we push to openshift internal registry
    When the resources are created using the crd
    Then We create template from yaml
    And Apply template with oc new-app
    Then Check for pod creation and state
  
  Scenario: Test jenkins CR creation
    Given Jenkins operator is running
    When we create the jenkins cr
    Then we check the jenkins pod health

  

