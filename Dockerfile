# Build the manager binary
ARG BASE_IMAGE=
FROM $BASE_IMAGE as builder

WORKDIR /workspace

COPY hack/boilerplate.go.txt hack/boilerplate.go.txt

# Copy the go source
COPY main.go main.go
COPY config/ config/
COPY apis/ apis/
COPY controllers/ controllers/

RUN make manifests generate fmt vet release

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.7-1085
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/manifest.yaml .
COPY jsons ./jsons/
USER 65534:65534

ENTRYPOINT ["/manager"]
