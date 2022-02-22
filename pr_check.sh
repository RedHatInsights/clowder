#!/bin/bash

go version

echo "$MINIKUBE_SSH_KEY" > minikube-ssh-ident

while read line; do
    if [ ${#line} -ge 100 ]; then
        echo "Commit messages are limited to 100 characters."
        echo "The following commit message has ${#line} characters."
        echo "${line}"
        exit 1
    fi
done <<< "$(git log --pretty=format:%s $(git merge-base master HEAD)..HEAD)"

set -exv

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

export IMAGE_TAG=`git rev-parse --short HEAD`
export IMAGE_NAME=quay.io/cloudservices/clowder

export GOROOT="/opt/go/1.16.10"
export PATH="${GOROOT}/bin:${PATH}"

make envtest
make update-version

KUBEBUILDER_ASSETS=`bin/setup-envtest use 1.22 -p path ` go test ./... -coverprofile cover.out
mkdir -p testbin/bin
cp `bin/setup-envtest use 1.22 -p path `/* testbin/bin -r
CLOWDER_VERSION=`git describe --tags`

IMG=$IMAGE_NAME:$IMAGE_TAG make docker-build
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-push

docker rm clowdercopy || true
docker create --name clowdercopy $IMAGE_NAME:$IMAGE_TAG
docker cp clowdercopy:/manifest.yaml .
docker rm clowdercopy || true

CONTAINER_NAME="clowder-pr-check-$ghprbPullId"
docker rm -f CONTAINER_NAME
# NOTE: Make sure this volume is mounted 'ro', otherwise Jenkins cannot clean up the workspace due to file permission errors
set +e
docker run -i \
    --name $CONTAINER_NAME \
    -v $PWD:/workspace:ro \
    -e IMAGE_NAME=$IMAGE_NAME \
    -e IMAGE_TAG=$IMAGE_TAG \
    -e QUAY_USER=$QUAY_USER \
    -e QUAY_TOKEN=$QUAY_TOKEN \
    -e MINIKUBE_HOST=$MINIKUBE_HOST \
    -e MINIKUBE_ROOTDIR=$MINIKUBE_ROOTDIR \
    -e MINIKUBE_USER=$MINIKUBE_USER \
    -e CLOWDER_VERSION=$CLOWDER_VERSION \
    quay.io/psav/clowder_pr_check:v2.5 \
    /workspace/build/pr_check_inner.sh
TEST_RESULT=$?

mkdir artifacts

docker cp $CONTAINER_NAME:/container_workspace/artifacts/ $PWD

docker rm -f $CONTAINER_NAME
set -e

exit $TEST_RESULT
