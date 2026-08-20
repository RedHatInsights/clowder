#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-db-dr-shared-secret-update"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-db-dr-shared-secret-update"
mkdir -p "${TMP_DIR}"

set -x

# Retry finding both generated secrets
for i in {1..30}; do
  kubectl get secret --namespace=test-db-dr-shared-secret-update owner-app && \
  kubectl get secret --namespace=test-db-dr-shared-secret-update consumer-app && break
  sleep 1
done

kubectl get secret --namespace=test-db-dr-shared-secret-update owner-app > /dev/null || { echo "owner-app secret not found"; exit 1; }
kubectl get secret --namespace=test-db-dr-shared-secret-update consumer-app > /dev/null || { echo "consumer-app secret not found"; exit 1; }

# Extract cdappconfig.json from owner-app
kubectl get secret --namespace=test-db-dr-shared-secret-update owner-app -o json > ${TMP_DIR}/owner-secret-before.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/owner-secret-before.json | base64 -d > ${TMP_DIR}/owner-cdappconfig-before.json

# Verify owner-app initial database values
jq -r '.database.hostname == "owner-app.rds.example.com"' -e < ${TMP_DIR}/owner-cdappconfig-before.json
jq -r '.database.password == "initial-password"' -e < ${TMP_DIR}/owner-cdappconfig-before.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/owner-cdappconfig-before.json

# Save owner-app hashCache
jq -r '.hashCache' -e < ${TMP_DIR}/owner-cdappconfig-before.json > ${TMP_DIR}/hash-cache-owner-before

# Extract cdappconfig.json from consumer-app
kubectl get secret --namespace=test-db-dr-shared-secret-update consumer-app -o json > ${TMP_DIR}/consumer-secret-before.json
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/consumer-secret-before.json | base64 -d > ${TMP_DIR}/consumer-cdappconfig-before.json

# Verify consumer-app has same initial database values (shared from owner)
jq -r '.database.hostname == "owner-app.rds.example.com"' -e < ${TMP_DIR}/consumer-cdappconfig-before.json
jq -r '.database.password == "initial-password"' -e < ${TMP_DIR}/consumer-cdappconfig-before.json
jq -r '.database.username == "myappuser"' -e < ${TMP_DIR}/consumer-cdappconfig-before.json

# Save consumer-app hashCache
jq -r '.hashCache' -e < ${TMP_DIR}/consumer-cdappconfig-before.json > ${TMP_DIR}/hash-cache-consumer-before
