#@follow_tag(openshift/golang-builder:1.15)
FROM openshift/golang-builder:1.15 AS builder
ENV BUILD_VERSION=v4.6.0
ENV OS_GIT_MAJOR=4
ENV OS_GIT_MINOR=6
ENV OS_GIT_PATCH=0
ENV SOURCE_GIT_URL=@@OPERATOR_REPO@@
ENV SOURCE_GIT_COMMIT=@@OPERATOR_COMMIT@@

# Basics
ENV GIT_COMMITTER_NAME="OpenShift Developer Sevices"
ENV GIT_COMMITTER_EMAIL=openshift-dev-services+jenkins@redhat.com
ENV LANG=en_US.utf8

WORKDIR /go/src/github.com/jenkinsci/jenkins-automation-operator
COPY ${REMOTE_SOURCE}/app .

RUN make bin

FROM registry.redhat.io/ubi8/ubi-minimal
LABEL io.k8s.display-name="Jenkins Operator" \
      io.k8s.description="Jenkins Operator to deploy Jenkins continuous integration server" \
      io.openshift.tags="jenkins,jenkins2,ci" \
      com.redhat.delivery.appregistry="false" \
      maintainer="openshift-dev-services+jenkins@redhat.com" \
      License="GPLv2+" \
      name="redhat-developer/jenkins-operator" \
      com.redhat.component="jenkins-operator-container" \
      io.openshift.maintainer.product="Jenkins for OpenShift" \
      version="@@OPERATOR_VERSION@@" \
      summary="Jenkins Operator to deploy Jenkins continuous integration server"
LABEL A=B
ENV LANG=en_US.utf8 \
    OPERATOR_NAME=@@OPERATOR_NAME@@ \
    OPERATOR=/usr/local/bin/@@OPERATOR_NAME@@

COPY --from=builder /go/src/github.com/jenkinsci/jenkins-automation-operator/build/_output/bin/${OPERATOR_NAME} /jenkins-operator

ENTRYPOINT [ "/jenkins-operator" ]


