# Current Operator version
VERSION ?= 0.1.0

# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
ifeq ($(findstring -minikube,${MAKECMDGOALS}), -minikube)
IMG ?= 127.0.0.1:5000/clowder:$(shell git rev-parse --short=7 HEAD)
else
IMG ?= quay.io/cloudservices/clowder:$(shell git rev-parse --short=7 HEAD)
endif

# Use podman by default, docker as fallback
ifeq (,$(shell which podman))
$(info "no podman in $(PATH), using docker")
RUNTIME ?= docker
else
RUNTIME ?= podman
endif

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

api-docs:
	./build/build_api_docs.sh
	./build/build_config_docs.sh

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

release: manifests kustomize controller-gen
	cat build/prommie-operator-bundle.yaml > manifest.yaml
	cat config/crd/bases/cloud.redhat.com_clowdapps.yaml >> manifest.yaml
	cat config/crd/bases/cloud.redhat.com_clowdenvironments.yaml >> manifest.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd ../..
	$(KUSTOMIZE) build config/default >> manifest.yaml

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

genconfig:
	cd controllers/cloud.redhat.com/config && gojsonschema -p config -o types.go schema.json

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	$(RUNTIME) build -f build/Dockerfile . -t ${IMG}

# Build the docker image
docker-build-no-test-quick:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o bin/manager-cgo main.go
	$(RUNTIME) build -f build/Dockerfile-local . -t ${IMG}

# Build the docker image
docker-build-no-test:
	$(RUNTIME) build -f build/Dockerfile . -t ${IMG}

# Push the docker image
docker-push:
	$(RUNTIME) push ${IMG}

# Push the docker image
docker-push-minikube:
	$(RUNTIME) push ${IMG} $(shell minikube ip):5000/clowder:$(shell git rev-parse --short=7 HEAD) --tls-verify=false

deploy-minikube: bundle-verify docker-build-no-test docker-push-minikube deploy

deploy-minikube-quick: bundle-verify docker-build-no-test-quick docker-push-minikube deploy


# find or download controller-gen
# download controller-gen if necessary
controller-gen:
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

kustomize:
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

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests bundle-verify
	$(RUNTIME) build -f bundle.Dockerfile -t $(BUNDLE_IMAGE):$(BUNDLE_IMAGE_TAG) .

bundle-verify:
	echo ${MAKECMDGOALS}
	echo ${IMG}
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
ifneq ($(origin REPLACE_VERSION), undefined)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(REPLACE_VERSION) $(BUNDLE_METADATA_OPTS)
endif
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
