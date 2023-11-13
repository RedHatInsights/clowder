#!/bin/bash

go version

set -exv

RESPONSE='$( \
    curl -Ls -H "Authorization: Bearer ${QUAY_TOKEN}" \
    "https://quay.io/api/v1/repository/cloudservices/clowder-base/tag/?specificTag=${BASE_TAG}" \
)'

echo "received HTTP response: ${RESPONSE}"

docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

docker build -f Dockerfile.test --build-arg BASE_IMAGE=${BASE_IMG} -t $TEST_CONTAINER .

docker run -i \
    -v `$PWD/bin/setup-envtest use -p path`:/bins:ro \
    -e IMAGE_NAME=${IMAGE_NAME} \
    -e IMAGE_TAG=${IMAGE_TAG} \
    -e QUAY_USER=$QUAY_USER \
    -e QUAY_TOKEN=$QUAY_TOKEN \
    -e MINIKUBE_HOST=$MINIKUBE_HOST \
    -e MINIKUBE_ROOTDIR=$MINIKUBE_ROOTDIR \
    -e MINIKUBE_USER=$MINIKUBE_USER \
    -e CLOWDER_VERSION=$CLOWDER_VERSION \
    $TEST_CONTAINER \
    make test
