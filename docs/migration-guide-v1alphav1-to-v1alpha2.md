# Migration guide from v1alpha1 to v1alpha2

## Stop jenkins-operator pod

Run command:
```bash
$ kubectl -n <namespace> scale deployment.apps/jenkins-operator --replicas=0
deployment.apps/jenkins-operator scaled
```

Desired state:
```bash
$ kubectl -n <namespace> get po
No resources found.
```

## Stop Jenkins master pod

Run command:
```bash
$ kubectl -n <namespace> get po
NAME                       READY     STATUS        RESTARTS   AGE
jenkins-operator-<cr_name>   2/2       Running       0          3m35s
$ kubectl -n <namespace> get delete po jenkins-operator-<cr_name>
pod "jenkins-operator-<cr_name>" deleted
```

Desired state:
```bash
$ kubectl -n <namespace> get po
No resources found.
```

## Save Jenkins CR to jenkins.yaml file

Run command:
```bash
$ kubectl -n <namespace> get jenkins <cr_name> -o yaml > jenkins.yaml
```

## Modify jenkins.yaml file

Old format:
```yaml
apiVersion: jenkins.io/v1alpha1
kind: Jenkins
metadata:
  name: <cr_name>
  namespace: <namespace>
spec:
  master:
    basePlugins:
      configuration-as-code:1.17:
      - configuration-as-code-support:1.17
      git:3.10.0:
      - apache-httpcomponents-client-4-api:4.5.5-3.0
      - credentials:2.1.19
      - display-url-api:2.3.1
      - git-client:2.7.7
      - jsch:0.1.55
      - junit:1.28
      - mailer:1.23
      - matrix-project:1.14
      - scm-api:2.4.1
      - script-security:1.59
      - ssh-credentials:1.16
      - structs:1.19
      - workflow-api:2.34
      - workflow-scm-step:2.7
      - workflow-step-api:2.19
      job-dsl:1.74:
      - script-security:1.59
      - structs:1.19
      kubernetes-credentials-provider:0.12.1:
      - credentials:2.1.19
      - structs:1.19
      - variant:1.2
      kubernetes:1.15.5:
      - apache-httpcomponents-client-4-api:4.5.5-3.0
      - cloudbees-folder:6.8
      - credentials:2.1.19
      - durable-task:1.29
      - jackson2-api:2.9.9
      - kubernetes-credentials:0.4.0
      - plain-credentials:1.5
      - structs:1.19
      - variant:1.2
      - workflow-step-api:2.19
      workflow-aggregator:2.6:
      - ace-editor:1.1
      - apache-httpcomponents-client-4-api:4.5.5-3.0
      - authentication-tokens:1.3
      - branch-api:2.5.2
      - cloudbees-folder:6.8
      - credentials-binding:1.18
      - credentials:2.1.19
      - display-url-api:2.3.1
      - docker-commons:1.15
      - docker-workflow:1.18
      - durable-task:1.29
      - git-client:2.7.7
      - git-server:1.7
      - handlebars:1.1.1
      - jackson2-api:2.9.9
      - jquery-detached:1.2.1
      - jsch:0.1.55
      - junit:1.28
      - lockable-resources:2.5
      - mailer:1.23
      - matrix-project:1.14
      - momentjs:1.1.1
      - pipeline-build-step:2.9
      - pipeline-graph-analysis:1.10
      - pipeline-input-step:2.10
      - pipeline-milestone-step:1.3.1
      - pipeline-model-api:1.3.8
      - pipeline-model-declarative-agent:1.1.1
      - pipeline-model-definition:1.3.8
      - pipeline-model-extensions:1.3.8
      - pipeline-rest-api:2.11
      - pipeline-stage-step:2.3
      - pipeline-stage-tags-metadata:1.3.8
      - pipeline-stage-view:2.11
      - plain-credentials:1.5
      - scm-api:2.4.1
      - script-security:1.59
      - ssh-credentials:1.16
      - structs:1.19
      - workflow-api:2.34
      - workflow-basic-steps:2.16
      - workflow-cps-global-lib:2.13
      - workflow-cps:2.69
      - workflow-durable-task-step:2.30
      - workflow-job:2.32
      - workflow-multibranch:2.21
      - workflow-scm-step:2.7
      - workflow-step-api:2.19
      - workflow-support:3.3
      workflow-job:2.32:
      - scm-api:2.4.1
      - script-security:1.59
      - structs:1.19
      - workflow-api:2.34
      - workflow-step-api:2.19
      - workflow-support:3.3
    image: jenkins/jenkins:lts
    imagePullPolicy: Always
    livenessProbe:
      failureThreshold: 12
      httpGet:
        path: /login
        port: 8080
        scheme: HTTP
      initialDelaySeconds: 30
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 5
    plugins:
      simple-theme-plugin:0.5.1: []
      slack:2.24:
      - workflow-step-api:2.19
      - credentials:2.1.19
      - display-url-api:2.3.1
      - junit:1.28
      - plain-credentials:1.5
      - script-security:1.59
      - structs:1.19
      - token-macro:2.8
    readinessProbe:
      failureThreshold: 12
      httpGet:
        path: /login
        port: 8080
        scheme: HTTP
      initialDelaySeconds: 30
      periodSeconds: 10
      successThreshold: 1
      timeoutSeconds: 5
    resources:
      limits:
        cpu: 1500m
        memory: 3Gi
      requests:
        cpu: "1"
        memory: 500Mi
```

New format:
```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: <cr_name>
  namespace: <namespace>
spec:
  master:
    basePlugins:
    - name: kubernetes
      version: 1.15.7
    - name: workflow-job
      version: "2.32"
    - name: workflow-aggregator
      version: "2.6"
    - name: git
      version: 3.10.0
    - name: job-dsl
      version: "1.74"
    - name: configuration-as-code
      version: "1.19"
    - name: configuration-as-code-support
      version: "1.19"
    - name: kubernetes-credentials-provider
      version: 0.12.1
    containers:
    - image: jenkins/jenkins:lts
      imagePullPolicy: Always
      livenessProbe:
        failureThreshold: 12
        httpGet:
          path: /login
          port: http
          scheme: HTTP
        initialDelaySeconds: 30
        periodSeconds: 10
        successThreshold: 1
        timeoutSeconds: 5
      name: jenkins-master
      readinessProbe:
        failureThreshold: 3
        httpGet:
          path: /login
          port: http
          scheme: HTTP
        initialDelaySeconds: 30
        periodSeconds: 10
        successThreshold: 1
        timeoutSeconds: 1
      resources:
        limits:
          cpu: 1500m
          memory: 3Gi
        requests:
          cpu: "1"
          memory: 500Mi
    plugins:
    - name: simple-theme-plugin
      version: 0.5.1
    - name: slack
      version: 2.24
```

Change apiVersion to `apiVersion: jenkins.io/v1alpha2`

New plugin format without dependent plugins:
- spec.master.basePlugins
- spec.master.plugins

Move Jenkins master container properties to spec.master.containers[jenkins-master]
- spec.master.image
- spec.master.imagePullPolicy
- spec.master.livenessProbe
- spec.master.readinessProbe
- spec.master.resources

## Deploy new Kubernetes manifests

Run commands:
```bash
$ kubectl -n <namespace> apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/crds/jenkins_v1alpha2_jenkins_crd.yaml
$ kubectl -n <namespace> apply -f jenkins.yaml
$ kubectl -n <namespace> apply -f https://raw.githubusercontent.com/jenkinsci/kubernetes-operator/master/deploy/all-in-one-v1alpha2.yaml
```