#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Test configuration
TEST_NAME="test-basic-app"
NAMESPACE="test-basic-app"
APP_NAME="puptoo"

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

# Run assertions
jq -r '.webPort == 8000' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.metricsPort == 9000' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.metricsPath == "/metrics"' -e < "${TMP_DIR}/${TEST_NAME}-json"

jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .hostname == "puptoo-processor.test-basic-app.svc"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .hostname == "puptoo-processor2.test-basic-app.svc"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .port == 8000' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .port == 8000' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPath == "/api/puptoo-processor/"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPaths[0] == "/api/puptoo-processor/"' -e < "${TMP_DIR}/${TEST_NAME}-json"

jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor") | .hostname == "puptoo-processor.test-basic-app.svc"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .hostname == "puptoo-processor2.test-basic-app.svc"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor") | .port == 10000' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .port == 10000' -e < "${TMP_DIR}/${TEST_NAME}-json"

echo "All assertions passed!"
