#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-config-secret-restarter"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-config-secret-restarter"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sh wait_for_generation.sh puptoo-processor "2"
kubectl get secret --namespace=test-config-secret-restarter puptoo -o json > ${TMP_DIR}/test-config-secret-restarter
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-config-secret-restarter | base64 -d > ${TMP_DIR}/test-config-secret-restarter-json
jq -r '.hashCache == "8c851988a320473f01d513a34f78e4640ea6fb788b2cb3d0db742b23d46f2c53e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < ${TMP_DIR}/test-config-secret-restarter-json
