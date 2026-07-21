#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Test configuration
TEST_NAME="test-v2-endpoints-clowdapp-authenticated-override"
NAMESPACE="test-v2-clowdapp-auth-override"
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

# Verify V2 public endpoint has authenticated=true (overridden from ClowdApp default of false)
jq -r '.dependencyEndpoints.v2.rbac.service.uri == "http://rbac-service.test-v2-clowdapp-auth-override.svc:8000"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.dependencyEndpoints.v2.rbac.service.authenticated == true' -e < "${TMP_DIR}/${TEST_NAME}-json"

# Verify V2 private endpoint has authenticated=true (overridden from ClowdApp default of false)
jq -r '.privateDependencyEndpoints.v2.rbac.service.uri == "http://rbac-service.test-v2-clowdapp-auth-override.svc:10000"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.privateDependencyEndpoints.v2.rbac.service.authenticated == true' -e < "${TMP_DIR}/${TEST_NAME}-json"

echo "All assertions passed!"
