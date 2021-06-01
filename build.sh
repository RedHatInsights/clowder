#!/bin/bash

set -x

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o bin/manager-cgo main.go

TAG=$(uuid | cut -f 1 -d -)
IMG=quay.io/klape/clowder:$TAG
podman build -f build/Dockerfile-local . -t $IMG
podman push $IMG $(minikube ip):5000/clowder:$TAG --tls-verify=false
cd config/manager && kustomize edit set image controller=127.0.0.1:5000/clowder:$TAG
cd ../..
kustomize build config/default | kubectl apply -f -
