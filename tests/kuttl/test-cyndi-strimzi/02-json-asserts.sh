#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-cyndi-strimzi"

# Test commands from original yaml file
sleep 5
kubectl get secret -n test-cyndi-strimzi-kafka test-cyndi-strimzi-host-inventory-db-cyndi -o json > /tmp/host-inventory-db-cyndi-secret
kubectl get secret -n test-cyndi-strimzi-kafka test-cyndi-strimzi-myapp-db-cyndi -o json > /tmp/myapp-db-cyndi-secret
kubectl get secret -n test-cyndi-strimzi host-inventory-db -o json > /tmp/host-inventory-db-secret
kubectl get secret -n test-cyndi-strimzi myapp-db -o json > /tmp/myapp-db-secret
kubectl get cyndipipeline -n test-cyndi-strimzi -o json > /tmp/cyndipipeline
jq '.data' /tmp/host-inventory-db-secret > /tmp/host-inventory-db-secret-data
jq '.data' /tmp/host-inventory-db-cyndi-secret > /tmp/host-inventory-db-cyndi-secret-data
EXPECTED=$(jq '.data["hostname"]' /tmp/host-inventory-db-secret); jq -e '.data["db.host"] == '$EXPECTED /tmp/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["port"]' /tmp/host-inventory-db-secret); jq -e '.data["db.port"] == '$EXPECTED /tmp/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["name"]' /tmp/host-inventory-db-secret); jq -e '.data["db.name"] == '$EXPECTED /tmp/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["username"]' /tmp/host-inventory-db-secret); jq -e '.data["db.user"] == '$EXPECTED /tmp/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["password"]' /tmp/host-inventory-db-secret); jq -e '.data["db.password"] == '$EXPECTED /tmp/host-inventory-db-cyndi-secret
EXPECTED=$(jq '.data["hostname"]' /tmp/myapp-db-secret); jq -e '.data["db.host"] == '$EXPECTED /tmp/myapp-db-cyndi-secret
EXPECTED=$(jq '.data["port"]' /tmp/myapp-db-secret); jq -e '.data["db.port"] == '$EXPECTED /tmp/myapp-db-cyndi-secret
EXPECTED=$(jq '.data["name"]' /tmp/myapp-db-secret); jq -e '.data["db.name"] == '$EXPECTED /tmp/myapp-db-cyndi-secret
USER=$(jq -r '.data["db.user"]' /tmp/myapp-db-cyndi-secret | base64 -d); [ "$USER" = "cyndi" ]
PW=$(jq -r '.data["db.password"]' /tmp/myapp-db-cyndi-secret | base64 -d); [ "$PW" = "cyndi" ]
EXPECTED=$(jq '."spec".additionalFilters' /tmp/cyndipipeline); jq -e '."spec".additionalFilters == '$EXPECTED /tmp/cyndipipeline
