# build stage
FROM golang:$GO_VERSION-alpine3.11 AS build-stage
ARG OPERATOR_SDK_VERSION
ENV GO111MODULE=on
RUN apk --no-cache add git curl make \
    && curl -L https://github.com/operator-framework/operator-sdk/releases/download/v$OPERATOR_SDK_VERSION/operator-sdk-v$OPERATOR_SDK_VERSION-x86_64-linux-gnu -o /usr/local/bin/operator-sdk \
    && chmod +x /usr/local/bin/operator-sdk
ADD . /kubernetes-operator
RUN cd /kubernetes-operator && make build config=config.in-cluster.env

# run stage
FROM golang:1.13.1-alpine3.10 
RUN apk --no-cache add git bash curl make
COPY --from=build-stage /kubernetes-operator/build/_output/bin/jenkins-operator /usr/local/bin/jenkins-operator
CMD [ "/usr/local/bin/jenkins-operator" ]
