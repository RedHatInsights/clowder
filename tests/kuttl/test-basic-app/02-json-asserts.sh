#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Test configuration
TEST_NAME="test-basic-app"
NAMESPACE="test-basic-app"
APP_NAME="puptoo"

# Setup error handling
setup_error_handling "${TEST_NAME}" "${NAMESPACE}"

# Wait for secret to be created
for i in {1..10}; do
    kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" && break || sleep 1
done

# Extract config from secret
kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" -o json > "/tmp/${TEST_NAME}"
jq -r '.data["cdappconfig.json"]' < "/tmp/${TEST_NAME}" | base64 -d > "/tmp/${TEST_NAME}-json"

# Run assertions
jq -r '.webPort == 8000' -e < "/tmp/${TEST_NAME}-json"
jq -r '.metricsPort == 9000' -e < "/tmp/${TEST_NAME}-json"
jq -r '.metricsPath == "/metrics"' -e < "/tmp/${TEST_NAME}-json"

jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .hostname == "puptoo-processor.test-basic-app.svc"' -e < "/tmp/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .hostname == "puptoo-processor2.test-basic-app.svc"' -e < "/tmp/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .port == 8000' -e < "/tmp/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .port == 8000' -e < "/tmp/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPath == "/api/puptoo-processor/"' -e < "/tmp/${TEST_NAME}-json"
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPaths[0] == "/api/puptoo-processor/"' -e < "/tmp/${TEST_NAME}-json"

jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor") | .hostname == "puptoo-processor.test-basic-app.svc"' -e < "/tmp/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .hostname == "puptoo-processor2.test-basic-app.svc"' -e < "/tmp/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor") | .port == 10000' -e < "/tmp/${TEST_NAME}-json"
jq -r '.privateEndpoints[] | select(.app == "puptoo") | select(.name == "processor2") | .port == 10000' -e < "/tmp/${TEST_NAME}-json"

echo "All assertions passed!"
