# Jenkins Operator Roadmap 2020

The following are the improvements/fixes/features which we need to add to the Jenkins 
Operator.

## Goals

- Jenkins Instance created using Deployment instead of Pod
    - Would allow better integration with other Kubernetes tools 
    which use MutationWebhooks for updating deployments based on their use-case.
    eg: Istio
- JenkinsImage Controller
    - A controller which will aid in the building of Jenkins Images using Kaniko so that
    plugins won't have be downloaded during runtime.
- Modularize User Reconciliation
    - This will include having individual controllers for CasC, SeedJobs and Backup/Restore
    to have fine grained control and a more decoupled code base.
- Refactor Jenkins CR reconciliation
    - After the above goal has been completed, this one will focus on having a better user
    experience with the Jenkins CR and removing the paradigm of the CR being the single 
    source of truth.
- Openshift Support 
    - Support for Openshift Jenkins Image, Route and Imagestream resources.
- Air-gapped Environment Support
    - Install and use Jenkins in an air-gapped environment wit multiple options to achieve this
    - Local Update Center
        - Created by a CR and would consume a PV where the plugins would be stored
    - JenkinsImage Controller
- Multibranch Pipeline Support 
