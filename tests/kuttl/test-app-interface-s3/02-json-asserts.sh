#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-app-interface-s3"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-app-interface-s3"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-app-interface-s3 puptoo && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-app-interface-s3 puptoo > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-app-interface-s3 puptoo -o json > ${TMP_DIR}/test-app-interface-s3
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-app-interface-s3 | base64 -d > ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .region == "us-east"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .accessKey == "aws_access_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .name == "test-app-interface-s3"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .secretKey == "aws_secret_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .requestedName == "test-app-interface-s3"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .name == "test-iam-s3"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .accessKey == "aws_access_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .secretKey == "aws_secret_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .name == "test-iam-s3-2"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .accessKey == "aws_access_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .secretKey == "aws_secret_key"' -e < ${TMP_DIR}/test-app-interface-s3-json
