FROM registry.access.redhat.com/hi/go:1.26.11-builder-1794621086@sha256:2629b9713d8e4a9d4726742f9d45ba9b39f14580fa7236b541d1cd9c1967cbf3 AS builder

WORKDIR /app

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

RUN make manifests generate fmt vet

# Build
RUN go build -o manager main.go

FROM registry.access.redhat.com/hi/core-runtime:2.43-1794282318@sha256:81280d7fd6afa7a3d2a6da2e90e545cf8e5d187c80a69b2a1c875fdcd731aaed

COPY --from=builder /app/manager /manager
COPY jsons ./jsons/

ENTRYPOINT ["/manager"]
