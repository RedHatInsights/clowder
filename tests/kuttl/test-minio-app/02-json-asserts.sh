#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-minio-app"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-minio-app"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-minio-app puptoo && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-minio-app puptoo > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-minio-app puptoo -o json > ${TMP_DIR}/test-minio-app
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-minio-app | base64 -d > ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.buckets[] | select(.requestedName == "first-bucket") | .name == "first-bucket"' -e < ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.buckets[] | select(.requestedName == "second-bucket") | .name == "second-bucket"' -e < ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.hostname == "test-minio-app-minio.test-minio-app.svc"' -e < ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.port == 9000' -e < ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.accessKey != ""' -e < ${TMP_DIR}/test-minio-app-json
jq -r '.objectStore.secretKey != ""' -e < ${TMP_DIR}/test-minio-app-json
