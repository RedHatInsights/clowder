FROM registry.access.redhat.com/ubi8/go-toolset:1.21.11-8.1724662611 as builder
USER 0
RUN dnf install -y openssh-clients git make which jq python3

COPY ci/minikube_e2e_tests_inner.sh .
RUN chmod 775 minikube_e2e_tests_inner.sh

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

RUN go mod download

COPY Makefile Makefile

RUN make controller-gen kustomize

RUN GO111MODULE=on go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.8.0 \
    && GO111MODULE=on go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.2

COPY hack/boilerplate.go.txt hack/boilerplate.go.txt

COPY main.go main.go
COPY config/ config/
COPY apis/ apis/
COPY controllers/ controllers/

RUN make manifests generate fmt vet release

RUN rm main.go
RUN rm -rf config
RUN rm -rf apis
RUN rm -rf controllers

# Build the manager binary
#ARG BASE_IMAGE=
#FROM $BASE_IMAGE as builder

WORKDIR /workspace

COPY hack/boilerplate.go.txt hack/boilerplate.go.txt

# Copy the go source
COPY main.go main.go
COPY config/ config/
COPY apis/ apis/
COPY controllers/ controllers/

RUN make manifests generate fmt vet release

# Build
RUN CGO_ENABLED=1 GOOS=linux GO111MODULE=on go build -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1052.1724178568
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/manifest.yaml .
COPY jsons ./jsons/
USER 65534:65534

ENTRYPOINT ["/manager"]
