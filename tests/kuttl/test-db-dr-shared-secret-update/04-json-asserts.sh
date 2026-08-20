#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-db-dr-shared-secret-update"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-db-dr-shared-secret-update"
mkdir -p "${TMP_DIR}"

set -x

# Wait for both deployments to roll (generation bump from 1 to 2)
sh ../_common/wait_for_generation.sh owner-app-service "2" test-db-dr-shared-secret-update
sh ../_common/wait_for_generation.sh consumer-app-service "2" test-db-dr-shared-secret-update

# Extract cdappconfig.json from owner-app after mutation
kubectl get secret --namespace=test-db-dr-shared-secret-update owner-app -o json > ${TMP_DIR}/owner-secret-after.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/owner-secret-after.json | base64 -d > ${TMP_DIR}/owner-cdappconfig-after.json

# Verify owner-app updated database values
jq -r '.database.hostname == "owner-app-restored.rds.example.com"' -e < ${TMP_DIR}/owner-cdappconfig-after.json
jq -r '.database.password == "restored-password"' -e < ${TMP_DIR}/owner-cdappconfig-after.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/owner-cdappconfig-after.json

# Verify owner-app hashCache changed
jq -r '.hashCache' -e < ${TMP_DIR}/owner-cdappconfig-after.json > ${TMP_DIR}/hash-cache-owner-after
diff ${TMP_DIR}/hash-cache-owner-before ${TMP_DIR}/hash-cache-owner-after > /dev/null || exit 0 && exit 1

# Extract cdappconfig.json from consumer-app after mutation
kubectl get secret --namespace=test-db-dr-shared-secret-update consumer-app -o json > ${TMP_DIR}/consumer-secret-after.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/consumer-secret-after.json | base64 -d > ${TMP_DIR}/consumer-cdappconfig-after.json

# Verify consumer-app also got updated database values
jq -r '.database.hostname == "owner-app-restored.rds.example.com"' -e < ${TMP_DIR}/consumer-cdappconfig-after.json
jq -r '.database.password == "restored-password"' -e < ${TMP_DIR}/consumer-cdappconfig-after.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/consumer-cdappconfig-after.json

# Verify consumer-app hashCache also changed
jq -r '.hashCache' -e < ${TMP_DIR}/consumer-cdappconfig-after.json > ${TMP_DIR}/hash-cache-consumer-after
diff ${TMP_DIR}/hash-cache-consumer-before ${TMP_DIR}/hash-cache-consumer-after > /dev/null || exit 0 && exit 1
