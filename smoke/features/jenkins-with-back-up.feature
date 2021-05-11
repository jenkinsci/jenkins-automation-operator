Feature: Jenkins with backup enabled

  As a user of Jenkins Operator
      I want to have a jenkins instance with backup enabled
      
Background:
    Given Project [TEST_NAMESPACE] is used

Scenario: Create jenkins with backup
    Given All containers in the jenkins pod are running
    When we check for the default backupconfig
    Then We create backup object using backup.yaml
    And We check for the backup object named example
    Then We rsh into the backup container and check for the jenkins-backups folder contents