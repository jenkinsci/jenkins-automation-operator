
GOLANGCI_LINT_CACHE := $(shell pwd)/build/_output/golangci-lint-cache
XDG_CACHE_HOME := $(shell pwd)/build/_output/xdgcache
GOCACHE := $(shell pwd)/build/_output/gocache

GIT_COMMIT_ID ?= $(shell git rev-parse --short HEAD)
# Default bundle image tag
BUNDLE_IMG ?= quay.io/redhat-developer/openshift-jenkins-operator-bundle:$(OPERATOR_VERSION)-$(GIT_COMMIT_ID)
# Image URL to use all building/pushing image targets
OPERATOR_IMG ?= quay.io/redhat-developer/openshift-jenkins-operator:$(OPERATOR_VERSION)-$(GIT_COMMIT_ID)

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)


# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

XDG_CACHE_HOME := $(shell pwd)/build/_output/xdgcache
GOOS := $(shell go env GOOS)
GOARCH :=  $(shell go env GOARCH)
