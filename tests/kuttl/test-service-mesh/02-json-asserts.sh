#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-service-mesh"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-service-mesh"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-service-mesh puptoo && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-service-mesh puptoo > /dev/null || { echo "Secret not found after retries"; exit 1; }
kubectl get secret --namespace=test-service-mesh puptoo -o json > ${TMP_DIR}/test-service-mesh
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-service-mesh | base64 -d > ${TMP_DIR}/test-service-mesh-json
jq -r '.webPort == 8000' -e < ${TMP_DIR}/test-service-mesh-json
jq -r '.metricsPort == 9000' -e < ${TMP_DIR}/test-service-mesh-json
jq -r '.metricsPath == "/metrics"' -e < ${TMP_DIR}/test-service-mesh-json
