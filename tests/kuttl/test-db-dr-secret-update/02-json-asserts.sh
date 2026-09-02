#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-db-dr-secret-update"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-db-dr-secret-update"
mkdir -p "${TMP_DIR}"

set -x

# Retry finding the secret
for i in {1..30}; do
  kubectl get secret --namespace=test-db-dr-secret-update myapp && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-db-dr-secret-update myapp > /dev/null || { echo "Secret not found"; exit 1; }

# Extract cdappconfig.json
kubectl get secret --namespace=test-db-dr-secret-update myapp -o json > ${TMP_DIR}/secret-before.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/secret-before.json | base64 -d > ${TMP_DIR}/cdappconfig-before.json

# Verify initial database values
jq -r '.database.hostname == "myapp.rds.example.com"' -e < ${TMP_DIR}/cdappconfig-before.json
jq -r '.database.password == "initial-password"' -e < ${TMP_DIR}/cdappconfig-before.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/cdappconfig-before.json

# Save hashCache for later comparison
jq -r '.hashCache' -e < ${TMP_DIR}/cdappconfig-before.json > ${TMP_DIR}/hash-cache-before
