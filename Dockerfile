FROM registry.access.redhat.com/ubi9/go-toolset:1.26.5-1785791459 AS builder
USER 0
ENV GOSUMDB=off

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer

RUN go mod download

COPY Makefile Makefile

COPY hack/boilerplate.go.txt hack/boilerplate.go.txt

# Copy the go source
COPY main.go main.go
COPY config/ config/
COPY apis/ apis/
COPY controllers/ controllers/

RUN make manifests generate fmt vet release

# Build
RUN CGO_ENABLED=1 GOOS=linux GO111MODULE=on go build -o manager main.go

FROM registry.access.redhat.com/ubi9-minimal:9.8-1785777232
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/manifest.yaml .
COPY jsons ./jsons/
USER 65534:65534

ENTRYPOINT ["/manager"]
