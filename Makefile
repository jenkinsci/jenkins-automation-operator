include scripts/help.mk
include scripts/commons.mk
include scripts/golang-tools.mk

build: manager

verify: fmt vet lint ## Verifies code before commit (fmt, lint, ...)

bin: FORCE ## Builds operator binary
	go build -o build/_output/bin/jenkins-operator main.go

e2e: install-ginkgo ## Run end-to-end (e2e) tests only
	$(GOBIN)/ginkgo -v ./controllers/...

test: kubebuilder generate manifests ## Run tests
	go test ./... -coverprofile cover.out

manager: generate goimports fmt vet bin ## Build manager binary

run: generate fmt vet manifests install ## Run in the configured Kubernetes cluster in ~/.kube/config. Prepend WATCH_NAMESPACE for single namespace mode.
	go run ./main.go

install: manifests kustomize ## Install CRDs into a cluster
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from a cluster
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

teardown: manifests kustomize ## Teardown deployed controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl delete -f -

manifests: controller-gen ## Generate manifests e.g. CRD, RBAC etc.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=jenkins-operator-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

fmt: # Run go fmt against code : formats the code
	go fmt ./...

vet: # Run go vet against code : check bugs
	go vet ./...

lint: golangci goimports  # Run golangci-lint against code : check formattung and bugs

GOLANGCI_LINT_CACHE := $(shell pwd)/build/_output/golangci-lint-cache
XDG_CACHE_HOME := $(shell pwd)/build/_output/xdgcache
GOCACHE := $(shell pwd)/build/_output/gocache

golangci: install-golangci
	GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) XDG_CACHE_HOME=$(XDG_CACHE_HOME) GOCACHE=$(GOCACHE) $(GOBIN)/golangci-lint run
goimports: install-goimports
	@goimports -w -l -e $(shell find . -type f -name '*.go' -not -path "./vendor/*")

generate: controller-gen ## Generate code
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

image-build: test ## Build the container image
	docker build . -t ${IMG}

docker-push: ## Push the container image
	docker push ${IMG}

bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

bundle-push: ## Push the bundle image.
	docker push $(BUNDLE_IMG)

bundle: manifests ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

FORCE:
	@echo ""
## Refer to golang-tools.mk include for controller-gen, golangci-install, goimports-install targets
