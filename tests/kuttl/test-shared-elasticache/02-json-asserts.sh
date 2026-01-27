#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-shared-elasticache"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-shared-elasticache"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-shared-elasticache-ns2 another-app && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-shared-elasticache-ns2 another-app > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-shared-elasticache-ns2 another-app -o json > ${TMP_DIR}/test-shared-elasticache
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-shared-elasticache | base64 -d > ${TMP_DIR}/test-shared-elasticache-json
jq -r '.inMemoryDb.hostname == "lovely"' -e < ${TMP_DIR}/test-shared-elasticache-json
jq -r '.inMemoryDb.port == 6767' -e < ${TMP_DIR}/test-shared-elasticache-json
