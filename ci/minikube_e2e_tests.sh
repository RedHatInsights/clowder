#!/bin/bash

echo "$MINIKUBE_SSH_KEY" > minikube-ssh-ident

set -exv

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

IMG=$IMAGE_NAME:$IMAGE_TAG BASE_IMG=$BASE_IMG make docker-build
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-push

tempid=$(docker create "$IMAGE_NAME:$IMAGE_TAG")
docker cp "${tempid}:/manifest.yaml" .
docker rm $tempid

docker build -f Dockerfile.test --build-arg BASE_IMAGE=${BASE_IMG} -t $CONTAINER_NAME .

chmod -R 755 $PWD/workspace

set +e
docker run -i \
    --name $CONTAINER_NAME \
    -v $PWD:/workspace:ro \
    -v `$PWD/bin/setup-envtest use -p path`:/bins:ro \
    -e IMAGE_NAME=$IMAGE_NAME \
    -e IMAGE_TAG=$IMAGE_TAG \
    -e QUAY_USER=$QUAY_USER \
    -e QUAY_TOKEN=$QUAY_TOKEN \
    -e MINIKUBE_HOST=$MINIKUBE_HOST \
    -e MINIKUBE_ROOTDIR=$MINIKUBE_ROOTDIR \
    -e MINIKUBE_USER=$MINIKUBE_USER \
    -e CLOWDER_VERSION=$CLOWDER_VERSION \
    $BASE_IMG \
    ./ci/minikube_e2e_tests_inner.sh
TEST_RESULT=$?

docker cp $CONTAINER_NAME:/container_workspace/artifacts/ $PWD

set -e

exit $TEST_RESULT
