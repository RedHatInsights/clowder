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

python3 -m venv .venv
source .venv/bin/activate
pip3 install sphinx-rtd-theme

cd docs && make clean && make html && cd -

deactivate

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

export IMAGE_TAG=`git rev-parse --short HEAD`
export IMAGE_NAME=quay.io/cloudservices/clowder

export ENVTEST_ASSETS_DIR=$PWD/testbin
mkdir -p $ENVTEST_ASSETS_DIR
test -f $ENVTEST_ASSETS_DIR/setup-envtest.sh || curl -sSLo $ENVTEST_ASSETS_DIR/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
source $ENVTEST_ASSETS_DIR/setup-envtest.sh; fetch_envtest_tools $ENVTEST_ASSETS_DIR; setup_envtest_env $ENVTEST_ASSETS_DIR;

IMG=$IMAGE_NAME:$IMAGE_TAG make docker-build
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-push

CONTAINER_NAME="clowder-pr-check-$ghprbPullId"
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
    quay.io/psav/clowder_pr_check:v2.5 \
    /workspace/build/pr_check_inner.sh
TEST_RESULT=$?

mkdir artifacts

docker cp $CONTAINER_NAME:/container_workspace/artifacts/ $PWD

docker rm -f $CONTAINER_NAME
set -e

exit $TEST_RESULT
