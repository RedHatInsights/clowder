#!/bin/bash

set -exv

IMAGE="quay.io/cloudservices/clowder"
IMAGE_TAG=$(git rev-parse --short=7 HEAD)

if [[ -z "$QUAY_USER" || -z "$QUAY_TOKEN" ]]; then
    echo "QUAY_USER and QUAY_TOKEN must be set"
    exit 1
fi

if [[ -z "$RH_REGISTRY_USER" || -z "$RH_REGISTRY_TOKEN" ]]; then
    echo "RH_REGISTRY_USER and RH_REGISTRY_TOKEN  must be set"
    exit 1
fi

BASE_TAG=`cat go.mod go.sum Dockerfile.base | sha256sum  | head -c 7`
BASE_IMG=quay.io/cloudservices/clowder-base:$BASE_TAG

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker --config="$DOCKER_CONF" login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io
docker --config="$DOCKER_CONF" login -u="$RH_REGISTRY_USER" -p="$RH_REGISTRY_TOKEN" registry.redhat.io

RESPONSE=$( \
        curl -Ls -I -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $QUAY_API_TOKEN" \
        https://quay.io/api/v1/repository/cloudservices/clowder-base/tag/$BASE_TAG/images \
    )
    echo "received HTTP response: $RESPONSE"

if [[ $RESPONSE != 200 ]]; then
    docker --config="$DOCKER_CONF" build -f Dockerfile.base . -t "$BASE_IMG"
	docker --config="$DOCKER_CONF" push "$BASE_IMG"
fi

make update-version
docker --config="$DOCKER_CONF" build  --build-arg BASE_IMAGE="$BASE_IMG" -t "${IMAGE}:${IMAGE_TAG}" .
docker --config="$DOCKER_CONF" push "${IMAGE}:${IMAGE_TAG}"
