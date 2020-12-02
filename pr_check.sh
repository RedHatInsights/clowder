#!/bin/bash

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

curl -LO https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.3.1/kubebuilder_2.3.1_linux_amd64.tar.gz

tar xzvf kubebuilder_2.3.1_linux_amd64.tar.gz
export KUBEBUILDER_ASSETS=$PWD/kubebuilder_2.3.1_linux_amd64/bin

IMG=$IMAGE_NAME:$IMAGE_TAG make docker-build
IMG=$IMAGE_NAME:$IMAGE_TAG make docker-push

docker run -i -v $PWD:/srcroot -e IMAGE_NAME=$IMAGE_NAME -e IMAGE_TAG=$IMAGE_TAG -e QUAY_USER=$QUAY_USER -e QUAY_TOKEN=$QUAY_TOKEN -e MINIKUBE_HOST=$MINIKUBE_HOST -e MINIKUBE_ROOTDIR=$MINIKUBE_ROOTDIR -e MINIKUBE_USER=$MINIKUBE_USER quay.io/psav/clowder_pr_check:blarg24 /srcroot/pr_check_inner.sh
