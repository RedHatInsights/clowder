#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multiple-app-endpoints"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-multiple-app-endpoints"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding puptoo-a secret
for i in {1..60}; do
  kubectl get secret --namespace=test-multiple-app-endpoints puptoo-a && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-multiple-app-endpoints puptoo-a > /dev/null || { echo "Secret \"puptoo-a\" not found"; exit 1; }

kubectl get secret puptoo-a -o json -n test-multiple-app-endpoints > ${TMP_DIR}/test-multiple-app-endpoints
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multiple-app-endpoints | base64 -d > ${TMP_DIR}/test-multiple-app-endpoints-json

# Retry finding puptoo-b secret
for i in {1..60}; do
  kubectl get secret --namespace=test-multiple-app-endpoints-b puptoo-b && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-multiple-app-endpoints-b puptoo-b > /dev/null || { echo "Secret \"puptoo-b\" not found"; exit 1; }

kubectl get secret puptoo-b -o json -n test-multiple-app-endpoints-b > ${TMP_DIR}/test-multiple-app-endpoints-b
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multiple-app-endpoints-b | base64 -d > ${TMP_DIR}/test-multiple-app-endpoints-json-b
jq -r '.endpoints[] | select(.app == "puptoo-a") | .name == "processor"' -e < ${TMP_DIR}/test-multiple-app-endpoints-json
jq -r '.endpoints[] | select(.app == "puptoo-a-2") | .name == "processor"' -e < ${TMP_DIR}/test-multiple-app-endpoints-json
jq -r '.endpoints | length == 2' -e < ${TMP_DIR}/test-multiple-app-endpoints-json
jq -r '.endpoints[] | select(.app == "puptoo-b") | .name == "processor"' -e < ${TMP_DIR}/test-multiple-app-endpoints-json-b
jq -r '.endpoints[] | select(.app == "puptoo-b-2") | .name == "processor"' -e < ${TMP_DIR}/test-multiple-app-endpoints-json-b
jq -r '.endpoints | length == 2' -e < ${TMP_DIR}/test-multiple-app-endpoints-json-b
