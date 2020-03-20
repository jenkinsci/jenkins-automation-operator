# OpenShift

## Build in OpenShift

```
oc new-build https://github.com/jenkinsci/jenkins-operator.git --context-dir=build/ --build-arg OPERATOR_SDK_VERSION=0.15.1 -o json | \
jq 'del(.items[2].spec.source.contextDir) | .items[2].spec.strategy.dockerStrategy += { "dockerfilePath": "build/Dockerfile" } ' | \
oc create -f - 
```

## Run in OpenShift

```
oc create -f deploy/crds/jenkins_v1alpha2_jenkins_crd.yaml
oc create -f deploy/service_account.yaml
oc create -f deploy/role.yaml
oc policy add-role-to-user jenkins-operator -z jenkins-operator
oc new-app jenkins-operator -e WATCH_NAMESPACE="" -e OPERATOR_NAME=jenkins-operator -o json | \
jq '.items[0].spec.template.spec.containers[0].env[2] += { "name": "POD_NAME", "valueFrom": { "fieldRef": { "fieldPath": "metadata.name" }}}' | \
jq '.items[0].spec.template.spec += { "serviceAccount": "jenkins-operator" }' | oc create -f -
oc set serviceaccount  dc jenkins-operator jenkins-operator
```

## Testing on OpenShift

```

```
