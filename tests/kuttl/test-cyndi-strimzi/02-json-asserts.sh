#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-cyndi-strimzi"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-cyndi-strimzi"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sleep 5
kubectl get secret -n test-cyndi-strimzi-kafka test-cyndi-strimzi-host-inventory-db-cyndi -o json > ${TMP_DIR}/host-inventory-db-cyndi-secret
kubectl get secret -n test-cyndi-strimzi-kafka test-cyndi-strimzi-myapp-db-cyndi -o json > ${TMP_DIR}/myapp-db-cyndi-secret
kubectl get secret -n test-cyndi-strimzi host-inventory-db -o json > ${TMP_DIR}/host-inventory-db-secret
kubectl get secret -n test-cyndi-strimzi myapp-db -o json > ${TMP_DIR}/myapp-db-secret
kubectl get cyndipipeline -n test-cyndi-strimzi -o json > ${TMP_DIR}/cyndipipeline
jq '.data' ${TMP_DIR}/host-inventory-db-secret > ${TMP_DIR}/host-inventory-db-secret-data
jq '.data' ${TMP_DIR}/host-inventory-db-cyndi-secret > ${TMP_DIR}/host-inventory-db-cyndi-secret-data
EXPECTED=$(jq '.data["hostname"]' ${TMP_DIR}/host-inventory-db-secret); jq -e '.data["db.host"] == '$EXPECTED ${TMP_DIR}/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["port"]' ${TMP_DIR}/host-inventory-db-secret); jq -e '.data["db.port"] == '$EXPECTED ${TMP_DIR}/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["name"]' ${TMP_DIR}/host-inventory-db-secret); jq -e '.data["db.name"] == '$EXPECTED ${TMP_DIR}/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["username"]' ${TMP_DIR}/host-inventory-db-secret); jq -e '.data["db.user"] == '$EXPECTED ${TMP_DIR}/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["password"]' ${TMP_DIR}/host-inventory-db-secret); jq -e '.data["db.password"] == '$EXPECTED ${TMP_DIR}/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["hostname"]' ${TMP_DIR}/myapp-db-secret); jq -e '.data["db.host"] == '$EXPECTED ${TMP_DIR}/myapp-db-cyndi-secret
EXPECTED=$(jq '.data["port"]' ${TMP_DIR}/myapp-db-secret); jq -e '.data["db.port"] == '$EXPECTED ${TMP_DIR}/myapp-db-cyndi-secret
EXPECTED=$(jq '.data["name"]' ${TMP_DIR}/myapp-db-secret); jq -e '.data["db.name"] == '$EXPECTED ${TMP_DIR}/myapp-db-cyndi-secret
USER=$(jq -r '.data["db.user"]' ${TMP_DIR}/myapp-db-cyndi-secret | base64 -d); [ "$USER" = "cyndi" ]
PW=$(jq -r '.data["db.password"]' ${TMP_DIR}/myapp-db-cyndi-secret | base64 -d); [ "$PW" = "cyndi" ]
EXPECTED=$(jq '."spec".additionalFilters' ${TMP_DIR}/cyndipipeline); jq -e '."spec".additionalFilters == '$EXPECTED ${TMP_DIR}/cyndipipeline
