---
title: "AKS"
linkTitle: "AKS"
weight: 10
date: 2019-08-05
description: >
    Additional configuration for Azure Kubernetes Service
---

Azure AKS managed Kubernetes service adds to every pod the following envs:

```yaml
- name: KUBERNETES_PORT_443_TCP_ADDR
  value:
- name: KUBERNETES_PORT
  value: tcp://
- name: KUBERNETES_PORT_443_TCP
  value: tcp://
- name: KUBERNETES_SERVICE_HOST
  value:
```

The operator is aware of it and omits these envs when checking if Jenkins pod envs have been changed. It prevents 
restart Jenkins pod over and over again.
