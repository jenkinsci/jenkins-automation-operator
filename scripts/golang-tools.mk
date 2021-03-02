## Targets to install golang tools (golangci, goimports, etc...)

all: help

controller-gen: # find or download controller-gen
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

# find or download kustomize
kustomize: kubebuilder
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif


# find or download kubebuilder
kubebuilder:
ifeq (, $(shell which kubebuilder))
	@{ \
	set -e ; \
	KUBEBUILDER_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUBEBUILDER_TMP_DIR ;\
	curl -L https://go.kubebuilder.io/dl/2.3.1/$$GOOS/$$GOARCH | tar -xz -C $$KUBEBUILDER_TMP_DIR;\
	mkdir -p $$KUBEBUILDER_ASSETS; \
	mv kubebuilder_2.3.1_"$$GOOS"_"$$GOARCH"/bin/* $$KUBEBUILDER_ASSETS ;\
	rm -fr $$KUBEBUILDER_TMP_DIR ;\
	}
endif

# find or download ginkgo
install-ginkgo:
ifeq (, $(shell which ginkgo))
	@{ \
	set -e ;\
	GINKGO_TMP_DIR=$$(mktemp -d) ;\
	cd $$GINKGO_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/onsi/ginkgo/ginkgo ;\
	go get github.com/onsi/gomega/... ;\
	rm -rf $$GINKGO_TMP_DIR ;\
	}
endif

# find or download golangci
install-golangci:
ifeq (, $(shell which golangci-lint))
	@{ \
	set -e ;\
	GOLANGCI_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLANGCI_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0 ;\
	go get -u  mvdan.cc/gofumpt ;\
	go get github.com/daixiang0/gci ;\
	rm -rf $$GOLANGCI_TMP_DIR ;\
	}
GOLANGCI=$(GOBIN)/golangci-lint
else
GOLANGCI=$(shell which golangci-lint)
endif

# find or download golangci
install-goimports: 
ifeq (, $(shell which goimports))
	@{ \
	set -e ;\
	GOIMPORTS_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOIMPORTS_TMP_DIR ;\
	go mod init tmp ;\
	go get golang.org/x/tools/cmd/goimports ;\
	rm -rf $$GOIMPORTS_TMP_DIR ;\
	}
GOIMPORTS=$(GOBIN)/goimports
else
GOIMPORTS=$(shell which goimports)
endif

# Set global env vars, especially needed for go cache
.EXPORT_ALL_VARIABLES:
GOLANGCI_LINT_CACHE := $(shell pwd)/build/_output/golangci-lint-cache
XDG_CACHE_HOME := $(shell pwd)/build/_output/xdgcache
GOCACHE := $(shell pwd)/build/_output/gocache
CGO_ENABLED := 0
KUBEBUILDER_ASSETS := /tmp/kubebuilder
