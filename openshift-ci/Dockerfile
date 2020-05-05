FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR_NAME=jenkins-operator \
    OPERATOR=/usr/local/bin/$OPERATOR_NAME

COPY build/_output/bin/${OPERATOR_NAME} ${OPERATOR}
ENTRYPOINT ["/usr/local/bin/jenkins-operator"]

