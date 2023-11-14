#!/bin/bash

set -exv

CICD_BOOTSTRAP_URL='https://raw.githubusercontent.com/RedHatInsights/cicd-tools/main/src/bootstrap.sh'
source <(curl -sSL "$CICD_BOOTSTRAP_URL") image_builder

get_base_image_tag() {

  local tag

  tag=$(cat "${BASE_IMAGE_FILES[@]}" | sha256sum | head -c 8)

  if ! _base_image_files_unchanged; then
    CICD_IMAGE_BUILDER_IMAGE_TAG="$tag"
    tag=$(cicd::image_builder::get_image_tag)
  fi

  echo -n "$tag"
}

base_image_tag_exists() {

  local tag="$1"
  local repository="cloudservices/clowder-base"

  #response=$(curl -Ls -H "Authorization: Bearer $QUAY_API_TOKEN" \
  #      "https://quay.io/api/v1/repository/${repository}/tag/?specificTag=$tag&onlyActiveTags=true")
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

_base_image_files_unchanged() {

  local target_branch=${ghprbTargetBranch:-master}

  git diff --quiet "${BASE_IMAGE_FILES[@]}" "$target_branch"
}

build_main_image() {

  export CICD_IMAGE_BUILDER_BUILD_ARGS=("BASE_IMAGE=${BASE_IMAGE_NAME}:${BASE_IMAGE_TAG}")
  export CICD_IMAGE_BUILDER_IMAGE_NAME="quay.io/cloudservices/clowder"
  export CICD_IMAGE_BUILDER_IMAGE_TAG=$(git rev-parse --short=8 HEAD)

  local security_compliance_tag="sc-$(date +%Y%m%d)-$(git rev-parse --short=8 HEAD)"

  # If the "security-compliance" branch is used for the build, it will tag the image as such.
  if [[ $GIT_BRANCH == *"security-compliance"* ]]; then
    export CICD_IMAGE_BUILDER_ADDITIONAL_TAGS=("$security_compliance_tag")
  fi

  if ! cicd::image_builder::build --platform 'linux/arm64' --platform 'linux/amd64'; then
    cicd::log::error "Error building image for platform $platform"
    return 1
  fi
#  for platform in 'linux/amd64' 'linux/arm64'; do
#    if ! cicd::image_builder::build --platform "$platform"; then
#      cicd::log::error "Error building image for platform $platform"
#      return 1
#    fi
#  done

  local full_image_name="$(cicd::image_builder::get_full_image_name)"
  local manifests=("$full_image_name" "${full_image_name}-amd64" "${full_image_name}-arm64")

  cicd::container::cmd manifest create "${manifests[@]}"

  if ! cicd::image_builder::local_build; then
    cicd::image_builder::push
    cicd::container::cmd manifest push "$full_image_name"
  fi
}

BASE_IMAGE_FILES=("go.mod" "go.sum" "Dockerfile.base")
BASE_IMAGE_NAME='quay.io/cloudservices/clowder-base'
BASE_IMAGE_TAG=$(get_base_image_tag)

if base_image_tag_exists "$BASE_IMAGE_TAG"; then
  echo "Base image exists, skipping..."
else
  if ! build_base_image; then
    echo "Error building base image!"
    exit 1
  fi
fi

if ! make update-version; then
  echo "Error updating version!"
  exit 1
fi

if ! build_main_image; then
  echo "Error building image!"
  exit 1
fi
