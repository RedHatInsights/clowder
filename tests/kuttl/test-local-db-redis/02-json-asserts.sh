#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-local-db-redis"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-local-db-redis"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-local-db-redis app-a && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-local-db-redis app-a > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-local-db-redis app-a -o json > ${TMP_DIR}/kuttl/test-local-db-redis/test-local-db-redis-json-a
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/kuttl/test-local-db-redis/test-local-db-redis-json-a | base64 -d > ${TMP_DIR}/app-a-cdappconfig-json
jq -r '.inMemoryDb.hostname == "app-a-redis.test-local-db-redis.svc"' -e < ${TMP_DIR}/app-a-cdappconfig-json
jq -r '.inMemoryDb.port == 6379' -e < ${TMP_DIR}/app-a-cdappconfig-json
jq '.inMemoryDb | has("username")' < ${TMP_DIR}/app-a-cdappconfig-json | grep -q false
jq '.inMemoryDb | has("password")' < ${TMP_DIR}/app-a-cdappconfig-json | grep -q false
