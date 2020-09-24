#!/bin/sh
oc expose svc $(oc get svc --no-headers | grep http | cut -f1 -d\ ) &> /dev/null
oc get routes jenkins-example-http -o jsonpath='{  .spec .host }'

