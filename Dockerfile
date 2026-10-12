FROM registry.access.redhat.com/hi/go:1.26.8-1791559214 AS builder

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

FROM registry.access.redhat.com/hi/core-runtime:2.43-1791559214

COPY --from=builder /app/manager /manager
COPY jsons ./jsons/

ENTRYPOINT ["/manager"]
