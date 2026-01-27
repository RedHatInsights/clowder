#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ff-app-interface"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-ff-app-interface"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..5}; do
  kubectl get secret --namespace=test-ff-app-interface puptoo && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-ff-app-interface puptoo > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-ff-app-interface puptoo -o json > ${TMP_DIR}/test-ff-app-interface
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-ff-app-interface | base64 -d > ${TMP_DIR}/test-ff-app-interface-json
jq -r '.featureFlags.clientAccessToken == "app-b-stage.rds.example.com"' -e < ${TMP_DIR}/test-ff-app-interface-json
jq -r '.featureFlags.hostname == "test.featureflags.redhat.com"' -e < ${TMP_DIR}/test-ff-app-interface-json
jq -r '.featureFlags.port == 12345' -e < ${TMP_DIR}/test-ff-app-interface-json
jq -r '.featureFlags.scheme == "https"' -e < ${TMP_DIR}/test-ff-app-interface-json
