#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Test configuration
TEST_NAME="test-v2-endpoints-clowdapp-plaintext"
NAMESPACE="test-v2-clowdapp-plaintext"
APP_NAME="consumer"

# Setup error handling
setup_error_handling "${TEST_NAME}"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/${TEST_NAME}"
mkdir -p "${TMP_DIR}"

set -x

# Wait for secret to be created
for i in {1..10}; do
    kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" && break || sleep 1
done

# Extract config from secret
kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" -o json > "${TMP_DIR}/${TEST_NAME}"
jq -r '.data["cdappconfig.json"]' < "${TMP_DIR}/${TEST_NAME}" | base64 -d > "${TMP_DIR}/${TEST_NAME}-json"

# Run assertions - verify v2 endpoint structure (plaintext, in-cluster)
jq -r '.dependencyEndpoints.v2.rbac.service.uri == "http://rbac-service.test-v2-clowdapp-plaintext.svc:8000"' -e < "${TMP_DIR}/${TEST_NAME}-json"

echo "All assertions passed!"
