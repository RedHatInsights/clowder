#!/bin/bash

set -exv

CICD_BOOTSTRAP_URL='https://raw.githubusercontent.com/RedHatInsights/cicd-tools/main/src/bootstrap.sh'
# shellcheck source=/dev/null
source <(curl -sSL "$CICD_BOOTSTRAP_URL") image_builder

get_base_image_tag() {

  local tag

  tag=$(cat "${BASE_IMAGE_FILES[@]}" | sha256sum | head -c 8)

  if _base_image_files_changed; then
    CICD_IMAGE_BUILDER_IMAGE_TAG="$tag"
    tag=$(cicd::image_builder::get_image_tag)
  fi

  echo -n "$tag"
}

base_image_tag_exists() {

  local tag="$1"
  local repository="cloudservices/clowder-base"

  response=$(curl -sSL \
    "https://quay.io/api/v1/repository/${repository}/tag/?specificTag=${tag}&onlyActiveTags=true")

  echo "received HTTP response: ${response}"

  # find all non-expired tags
  [[ 1 -eq $(jq '.tags | length' <<<"$response") ]]
}

build_base_image() {

  export CICD_IMAGE_BUILDER_IMAGE_NAME="$BASE_IMAGE_NAME"
  export CICD_IMAGE_BUILDER_IMAGE_TAG="$BASE_IMAGE_TAG"
  export CICD_IMAGE_BUILDER_CONTAINERFILE_PATH="Dockerfile.base"

  cicd::image_builder::build_and_push
}

_base_image_files_changed() {

  local target_branch=${ghprbTargetBranch:-master}

  # Use git to check for any non staged differences in the Base Image files
  ! git diff --quiet "$target_branch" -- "${BASE_IMAGE_FILES[@]}"
}

build_main_image() {
  export CICD_IMAGE_BUILDER_CONTAINERFILE_PATH="Dockerfile"
  export CICD_IMAGE_BUILDER_BUILD_ARGS=("BASE_IMAGE=${BASE_IMAGE_NAME}:${BASE_IMAGE_TAG}")
  export CICD_IMAGE_BUILDER_IMAGE_NAME="quay.io/cloudservices/clowder"
  CICD_IMAGE_BUILDER_IMAGE_TAG=$(git rev-parse --short=8 HEAD)

  # If the "security-compliance" branch is used for the build, it will tag the image as such.
  if [[ "$GIT_BRANCH" == "origin/security-compliance" ]]; then
    CICD_IMAGE_BUILDER_IMAGE_TAG="sc-$(date +%Y%m%d)-${CICD_IMAGE_BUILDER_IMAGE_TAG}"
  fi

  export CICD_IMAGE_BUILDER_IMAGE_TAG

  cicd::image_builder::build_and_push
}

BASE_IMAGE_FILES=("go.mod" "go.sum")
#BASE_IMAGE_NAME='quay.io/cloudservices/clowder-base'
#BASE_IMAGE_TAG=$(get_base_image_tag)

#if base_image_tag_exists "$BASE_IMAGE_TAG"; then
#  echo "Base image exists, skipping..."
#else
#  if ! build_base_image; then
#    echo "Error building base image!"
#    exit 1
#  fi
#fi

if ! make update-version; then
  echo "Error updating version!"
  exit 1
fi

if ! build_main_image; then
  echo "Error building image!"
  exit 1
fi
