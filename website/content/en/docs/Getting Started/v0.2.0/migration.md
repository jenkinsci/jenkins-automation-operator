---
title: "Migration from v0.1.1"
linkTitle: "Migration from v0.1.1"
weight: 10
date: 2019-08-05
description: >
    How to migrate from v0.1.1 to v0.2.0
---

### Added seed job agent
Now seed jobs are not built by master executors, but by dedicated agent deployed into Kubernetes. We disabled master executors for security reasons.

### Apply Jenkins configuration via Groovy scripts instead of Jenkins jobs
We have removed hardcoded configuration by **Jenkins** jobs. 

In `v0.1.1` **Jenkins Operator** configuration was stored in `jenkins-operator-user-configuration-<cr_name>`
If you want to use `v0.2.0` or newer you must simply write refer to old ConfigMap by modifying CR, for example:

```yaml
apiVersion: jenkins.io/v1alpha2
kind: Jenkins
metadata:
  name: example
spec:
  configurationAsCode:
    configurations: 
    - name: jenkins-operator-user-configuration-<cr_name>
  groovyScripts:
    configurations:
    - name: jenkins-operator-user-configuration-<cr_name>
```

**Jenkins** configuration jobs (*Configure Seed Jobs*, *jenkins-operator-base-configuration*, *jenkins-operator-user-configuration*) have been removed from **Jenkins**.

In `v0.1.1` you can see if configuration failed or successfully updated in **Jenkins** UI (job build logs).
Now, when Jenkins configuration jobs are removed, you must use this command to see if configuration was failed.
```bash
$ kubectl -n logs deployment/jenkins-operator
```