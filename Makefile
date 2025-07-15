CLOWDER_BUILD_TAG ?= $(shell git rev-parse HEAD)

GO_CMD ?= go

OS := $(shell uname -s)

TEMPLATE_KUSTOMIZE ?= "deploy-kustomize.yaml"

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.30

# Image URL to use all building/pushing image targets
ifeq ($(findstring -minikube,${MAKECMDGOALS}), -minikube)
IMG ?= 127.0.0.1:5000/clowder:$(CLOWDER_BUILD_TAG)
else
IMG ?= quay.io/redhat-user-workloads/hcm-eng-prod-tenant/clowder/clowder:$(CLOWDER_BUILD_TAG)
endif

CLOWDER_VERSION ?= $(shell git describe --tags)

# Use podman by default, docker as fallback
ifeq (,$(shell which podman))
$(info "no podman in $(PATH), using docker")
RUNTIME ?= docker
else
RUNTIME ?= podman
endif

# Install gojsonschema if not found
GOJSONSCHEMA_BIN := $(shell which gojsonschema 2> /dev/null)
ifndef GOJSONSCHEMA_BIN
$(info gojsonschema binary not found. Installing...)
$(shell $(GO_CMD) install github.com/atombender/go-jsonschema/cmd/gojsonschema@v0.12.1)
$(info Ensure that $$GOPATH/bin is in your PATH.)
endif


KUTTL_TEST ?= ""

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO_CMD) env GOBIN))
GOBIN=$(shell $(GO_CMD) env GOPATH)/bin
else
GOBIN=$(shell $(GO_CMD) env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

api-docs:
	./build/build_api_docs.sh
	./build/build_config_docs.sh

build-template: 
	@echo "Checking for $(TEMPLATE_KUSTOMIZE)"
	@if [ ! -f $(TEMPLATE_KUSTOMIZE) ]; then \
		$(MAKE) build-template-kustomize; \
	fi
	TEMPLATE_KUSTOMIZE=$(TEMPLATE_KUSTOMIZE) ./build/build_template.sh

build-template-kustomize: manifests kustomize controller-gen
	$(KUSTOMIZE) build config/deployment-template > $(TEMPLATE_KUSTOMIZE)

test-template: build-template
	source ./build/template_check.sh

release: manifests kustomize controller-gen
	echo "---" > manifest.yaml
	cat config/manager/clowder_config.yaml >> manifest.yaml
	echo "---" >> manifest.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd ../..
	$(KUSTOMIZE) build config/release-manifest >> manifest.yaml

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd webhook paths="./apis/..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) rbac:roleName=manager-role paths="./controllers/..."

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./controllers/..."

fmt: ## Run go fmt against code.
	$(GO_CMD) fmt ./...

vet: ## Run go vet against code.
	$(GO_CMD) vet ./...

test: update-version manifests envtest generate fmt vet
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" CLOWDER_CONFIG_PATH=$(PROJECT_DIR)/test_config.json $(GO_CMD) test ./... -coverprofile cover.out

vscode-debug: update-version manifests envtest generate fmt vet
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" code .

# Run kuttl tests, make kuttl. Or pass in a test to run, make kuttl KUTTL_TEST="--test=testephemeral-gateway"
kuttl: manifests generate fmt vet envtest
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" kubectl kuttl test \
	--config tests/kuttl/kuttl-test.yaml \
	--manifest-dir config/crd/bases/ \
	tests/kuttl/ \
	$(KUTTL_TEST)

##@ Build

genconfig:
	cd controllers/cloud.redhat.com/config && gojsonschema -p config -o types.go schema.json

build: update-version generate fmt vet ## Build manager binary.
	$(GO_CMD) build -o bin/manager main.go

run: update-version manifests generate fmt vet ## Run a controller from your host.
	$(GO_CMD) run ./main.go

# Build the docker image
docker-build: update-version
	$(RUNTIME) build . -t ${IMG}

# Build the docker image
docker-build-no-test-quick: update-version
	CGO_ENABLED=0 GOOS=linux GO111MODULE=on $(GO_CMD) build -o bin/manager-cgo main.go
	$(RUNTIME) build -f build/Dockerfile-local . -t ${IMG}

# Build the docker image
docker-build-no-test:
	$(RUNTIME) build . -t ${IMG}

# Push the docker image
docker-push:
	$(RUNTIME) push ${IMG}


# For folks using Darwin for their OS, they need to use Docker for pushing a local Clowder image
# to Minikube.
ifeq ($(OS),Darwin)
docker-push-minikube:
	docker push ${IMG}
else
docker-push-minikube:
	$(RUNTIME) push ${IMG} $(shell minikube ip):5000/clowder:$(CLOWDER_BUILD_TAG) --tls-verify=false
endif

deploy-minikube: docker-build-no-test docker-push-minikube deploy

deploy-minikube-quick: docker-build-no-test-quick docker-push-minikube deploy

# we can't git ignore these files, but we want to avoid overwriting them
no-update:
	git fetch origin
	git checkout origin/master -- config/manager/kustomization.yaml \
								  controllers/cloud.redhat.com/version.txt \
								  config/manifests/bases/clowder.clusterserviceversion.yaml

##@ Deployment

pre-push: manifests generate genconfig build-template api-docs no-update

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/release-manifest | minikube kubectl -- apply --validate=false -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/release-manifest | kubectl delete -f -

update-version: ## Updates the version in the image
	$(shell echo -n $(CLOWDER_VERSION) > controllers/cloud.redhat.com/version.txt)
	echo "Building version: $(CLOWDER_VERSION)"

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.16.4

update-deps:
	KUSTOMIZE_VERSION=$(KUSTOMIZE_VERSION) CONTROLLER_TOOLS_VERSION=$(CONTROLLER_TOOLS_VERSION) ./deps/update_e2e_deps.sh

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.17

