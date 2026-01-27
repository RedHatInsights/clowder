#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-shared-local-db-redis"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-shared-local-db-redis"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..30}; do
  kubectl get secret --namespace=test-local-db-redis-shared app-b && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-local-db-redis-shared app-b > /dev/null || { echo "Secret not found"; exit 1; }

kubectl get secret --namespace=test-local-db-redis-shared app-b -o json > ${TMP_DIR}/test-local-db-redis-shared-json-b
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-local-db-redis-shared-json-b | base64 -d > ${TMP_DIR}/app-b-cdappconfig-json
jq -r '.inMemoryDb.hostname == "app-a-redis.test-local-db-redis-shared.svc"' -e < ${TMP_DIR}/app-b-cdappconfig-json
jq -r '.inMemoryDb.port == 6379' -e < ${TMP_DIR}/app-b-cdappconfig-json
