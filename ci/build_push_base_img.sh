#!/bin/bash

set -exv

DOCKER_CONF="$PWD/.docker"
mkdir -p "$DOCKER_CONF"
docker login -u="$QUAY_USER" -p="$QUAY_TOKEN" quay.io

RESPONSE=$( \
    curl -Ls -H "Authorization: Bearer $QUAY_API_TOKEN" \
    "https://quay.io/api/v1/repository/cloudservices/clowder-base/tag/?specificTag=$BASE_TAG" \
)

echo "received HTTP response: $RESPONSE"

# find all non-expired tags
VALID_TAGS_LENGTH=$(echo $RESPONSE | jq '[ .tags[] | select(.end_ts == null) ] | length')

# Check if Clowder's base image tag already exists
if [[ "$VALID_TAGS_LENGTH" -eq 0 ]]; then
    BASE_IMG=$BASE_IMG make docker-build-and-push-base
else
    echo "Base image has already been built, Quay image push skipped."
fi
