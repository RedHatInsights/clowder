#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-web-services"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-web-services"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-web-services puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-web-services puptoo -o json > ${TMP_DIR}/test-web-services
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-web-services | base64 -d > ${TMP_DIR}/test-web-services-json
jq -r '.publicPort == 8000' -e < ${TMP_DIR}/test-web-services-json
jq -r '.metricsPort == 9000' -e < ${TMP_DIR}/test-web-services-json
jq -r '.privatePort == 10000' -e < ${TMP_DIR}/test-web-services-json
jq -r '.metricsPath == "/metrics"' -e < ${TMP_DIR}/test-web-services-json
