include scripts/colors.mk
include scripts/commons.mk
include scripts/golang-tools.mk


# Current Operator version
OPERATOR_NAME ?= jenkins-operator
OPERATOR_VERSION ?= 0.7.3

## This makefile is self documented: To set comment, add ## after the target
help: ## Display this help message
	@echo "    ${BLACK}:: ${YELLOW}make${RESET} ${BLACK}::${RESET}"
	@echo "${YELLOW}-------------------------------------------------------------------------------------------------------${RESET}"
	@grep -E '^[a-zA-Z_0-9%-]+:.*?## .*$$' $(word 1,$(MAKEFILE_LIST)) | awk 'BEGIN {FS = ":.*?## "}; {printf "${TARGET_COLOR}%-30s${RESET} %s\n", $$1, $$2}'

e2e: install-ginkgo ## Run end-to-end (e2e) tests only
	$(GOBIN)/ginkgo -v ./controllers/...

test: kubebuilder generate manifests ## Run tests
	@echo "${BLUE}Running unit tests and computing coverage${RESET}"
	@go test ./... -coverprofile cover.out

operator: goimports fmt vet generate test bin ## Builds operator binary

bin: # Builds operator binary only 
	@echo "${BLUE}Building ${OPERATOR_NAME} ${YELLOW}binary${BLUE}${RESET}"
	@go build -o build/_output/bin/manager main.go

## Bundle and image related targets
operator-image: operator## Build the operator container image
	@echo "${BLUE}Building ${OPERATOR_NAME} ${YELLOW}container${BLUE} image${RESET}"
	@docker build . -t ${OPERATOR_IMG}

operator-image-push: ## Push the operator container image
	docker push ${OPERATOR_IMG}

bundle-image: ## Build the bundle image.
	@echo "${BLUE}Building ${OPERATOR_NAME} ${YELLOW}bundle${BLUE} image${RESET}"
	@docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

bundle-image-push: ## Push the bundle image.
	docker push $(BUNDLE_IMG)

bundle: manifests ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(OPERATOR_VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

## Local and remote cluster development
local-run: generate fmt vet manifests crd-install ## Run in the configured Kubernetes cluster in ~/.kube/config. Prepend WATCH_NAMESPACE for single namespace mode.
	go run ./main.go

crd-install: manifests kustomize ## Install CRDs into a cluster
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

crd-uninstall: manifests kustomize ## Uninstall CRDs from a cluster
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

remote-setup: manifests kustomize ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${OPERATOR_IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

remote-teardown: manifests kustomize ## Teardown deployed controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${OPERATOR_IMG}
	$(KUSTOMIZE) build config/default | kubectl delete -f -

manifests: controller-gen # Generate manifests e.g. CRD, RBAC etc.
	@echo "${BLUE}Generating bundle's manifest and associated files (CRD, samples, RBAC)${RESET}"
	@$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=jenkins-operator-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen manifests kustomize# Generate code
	@echo "${BLUE}Generating controllers and setting license headers in file${RESET}"
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Code quality related targets
verify: fmt vet lint ## Verifies code before commit (fmt, lint, ...)

lint: golangci goimports  # Run golangci-lint against code : check formatting and bugs

golangci: install-golangci # Run golang-ci checks
	@echo "${BLUE}Checking golancgi formatting and bugs${RESET}"
	@GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) XDG_CACHE_HOME=$(XDG_CACHE_HOME) GOCACHE=$(GOCACHE) $(GOBIN)/golangci-lint run

goimports: install-goimports # Optimize go imports
	@echo "${BLUE}Optimizing go imports${RESET}"
	@goimports -w -l -e $(shell find . -type f -name '*.go' -not -path "./vendor/*") >> build/build.log

fmt: # Run go fmt against code : formats the code
	@echo "${BLUE}Formatting go files${RESET}"
	@go fmt ./...

vet: # Run go vet against code : check bugs
	@echo "${BLUE}Checking code style and common bugs${RESET}"
	@go vet ./...

FORCE:
	@echo ""
## Refer to golang-tools.mk include for controller-gen, golangci-install, goimports-install targets
