#!/bin/bash

set -exv

IMAGE="quay.io/cloudservices/clowder"
IMAGE_TAG=$(git rev-parse --short=8 HEAD)
SECURITY_COMPLIANCE_TAG="sc-$(date +%Y%m%d)-$(git rev-parse --short=8 HEAD)"

if [[ -z "$QUAY_USER" || -z "$QUAY_TOKEN" ]]; then
    echo "QUAY_USER and QUAY_TOKEN must be set"
    exit 1
fi

if [[ -z "$RH_REGISTRY_USER" || -z "$RH_REGISTRY_TOKEN" ]]; then
    echo "RH_REGISTRY_USER and RH_REGISTRY_TOKEN  must be set"
    exit 1
fi

BASE_TAG=`cat go.mod go.sum Dockerfile.base | sha256sum  | head -c 8`
BASE_IMG=quay.io/cloudservices/clowder-base:$BASE_TAG

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker --config="$DOCKER_CONF" login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io
docker --config="$DOCKER_CONF" login -u="$RH_REGISTRY_USER" -p="$RH_REGISTRY_TOKEN" registry.redhat.io

RESPONSE=$( \
        curl -Ls -H "Authorization: Bearer $QUAY_API_TOKEN" \
        "https://quay.io/api/v1/repository/cloudservices/clowder-base/tag/?specificTag=$BASE_TAG" \
    )

echo "received HTTP response: $RESPONSE"

# find all non-expired tags
VALID_TAGS_LENGTH=$(echo $RESPONSE | jq '[ .tags[] | select(.end_ts == null) ] | length')

if [[ "$VALID_TAGS_LENGTH" -eq 0 ]]; then
    docker --config="$DOCKER_CONF" build -f Dockerfile.base . -t "$BASE_IMG"
	docker --config="$DOCKER_CONF" push "$BASE_IMG"
fi

# If the "security-compliance" branch is used for the build, it will tag the image as such.
if [[ "$GIT_BRANCH" == "origin/security-compliance" ]]; then
    IMAGE_TAG="$SECURITY_COMPLIANCE_TAG"
fi

make update-version
docker --config="$DOCKER_CONF"  build  --platform linux/amd64 --build-arg BASE_IMAGE="$BASE_IMG" -t "${IMAGE}:${IMAGE_TAG}-amd64" --push .
docker --config="$DOCKER_CONF"  build  --platform linux/arm64 --build-arg BASE_IMAGE="$BASE_IMG" -t "${IMAGE}:${IMAGE_TAG}-arm64" --push .

docker --config="$DOCKER_CONF" manifest create "${IMAGE}:${IMAGE_TAG}" \
    "${IMAGE}:${IMAGE_TAG}-amd64" \
    "${IMAGE}:${IMAGE_TAG}-arm64"

docker --config="$DOCKER_CONF" manifest push "${IMAGE}:${IMAGE_TAG}"
