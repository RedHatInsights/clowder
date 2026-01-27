#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ff-local"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-ff-local"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..15}; do
  kubectl get secret --namespace=test-ff-local puptoo && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-ff-local puptoo > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-ff-local puptoo -o json > ${TMP_DIR}/test-ff-local
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-ff-local | base64 -d > ${TMP_DIR}/test-ff-local-json
jq -r '.webPort == 8000' -e < ${TMP_DIR}/test-ff-local-json
jq -r '.metricsPort == 9000' -e < ${TMP_DIR}/test-ff-local-json
jq -r '.metricsPath == "/metrics"' -e < ${TMP_DIR}/test-ff-local-json
jq -r '.featureFlags.hostname == "test-ff-local-featureflags.test-ff-local.svc"' -e < ${TMP_DIR}/test-ff-local-json
jq -r '.featureFlags.port == 4242' -e < ${TMP_DIR}/test-ff-local-json
jq -r '.featureFlags.scheme == "http"' -e < ${TMP_DIR}/test-ff-local-json
