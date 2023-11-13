#!/bin/bash

set -exv

source <(curl -sSL "https://raw.githubusercontent.com/RedHatInsights/cicd-tools/main/src/bootstrap.sh") image_builder

tag_exists() {

  local tag="$1"
  local response valid_tags_length

  response=$(curl -sSL \
    "https://quay.io/api/v1/repository/cloudservices/clowder-base/tag/?specificTag=${tag}&onlyActiveTags=true")

  echo "received HTTP response: ${response}"

  # find all non-expired tags
  valid_tags_length=$(jq '.tags | length' <<<"$response")

  # Check if Clowder's base image tag already exists
  [[ "$valid_tags_length" -ge 1 ]]
}

get_base_tag() {
  cat go.mod go.sum Dockerfile.base | sha256sum | head -c 8
}

export CICD_IMAGE_BUILDER_IMAGE_NAME="quay.io/cloudservices/clowder-base"
export CICD_IMAGE_BUILDER_IMAGE_TAG=$(get_base_tag)
export CICD_IMAGE_BUILDER_CONTAINERFILE_PATH="Dockerfile.base"

# Check if Clowder's base image tag already exists
if ! tag_exists "$CICD_IMAGE_BUILDER_IMAGE_TAG"; then
  cicd::image_builder::build_and_push
else
  echo "Base image has already been built, Quay image push skipped."
fi
