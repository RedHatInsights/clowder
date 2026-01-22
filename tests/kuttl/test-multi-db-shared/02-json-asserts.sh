#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multi-db-shared"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-multi-db-shared"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-multi-db-shared app-c && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-multi-db-shared app-c -o json > ${TMP_DIR}/test-multi-db-shared
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-db-shared | base64 -d > ${TMP_DIR}/test-multi-db-shared-json
jq -r '.database.hostname == "test-multi-db-shared-db-v13.test-multi-db-shared.svc"' -e < ${TMP_DIR}/test-multi-db-shared-json
jq -r '.database.sslMode == "disable"' -e < ${TMP_DIR}/test-multi-db-shared-json
kubectl get secret --namespace=test-multi-db-shared app-a -o json > ${TMP_DIR}/test-multi-db-shared-a
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-db-shared-a | base64 -d > ${TMP_DIR}/test-multi-db-shared-a-json
jq -r '.database.hostname == "test-multi-db-shared-db-v12.test-multi-db-shared.svc"' -e < ${TMP_DIR}/test-multi-db-shared-a-json
jq -r '.database.sslMode == "disable"' -e < ${TMP_DIR}/test-multi-db-shared-a-json
