#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-db-dr-secret-update"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-db-dr-secret-update"
mkdir -p "${TMP_DIR}"

set -x

# Wait for the deployment to roll (generation bump from 1 to 2)
sh ../_common/wait_for_generation.sh myapp-service "2" test-db-dr-secret-update

# Extract cdappconfig.json after mutation
kubectl get secret --namespace=test-db-dr-secret-update myapp -o json > ${TMP_DIR}/secret-after.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/secret-after.json | base64 -d > ${TMP_DIR}/cdappconfig-after.json

# Verify updated database values
jq -r '.database.hostname == "myapp-restored.rds.example.com"' -e < ${TMP_DIR}/cdappconfig-after.json
jq -r '.database.password == "restored-password"' -e < ${TMP_DIR}/cdappconfig-after.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/cdappconfig-after.json

# Verify hashCache changed (proves deployment was bounced)
jq -r '.hashCache' -e < ${TMP_DIR}/cdappconfig-after.json > ${TMP_DIR}/hash-cache-after
diff ${TMP_DIR}/hash-cache-before ${TMP_DIR}/hash-cache-after > /dev/null || exit 0 && exit 1
