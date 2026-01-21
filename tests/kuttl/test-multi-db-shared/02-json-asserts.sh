#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multi-db-shared" "test-multi-db-shared"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-multi-db-shared app-c && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-multi-db-shared app-c -o json > /tmp/test-multi-db-shared
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-db-shared | base64 -d > /tmp/test-multi-db-shared-json
jq -r '.database.hostname == "test-multi-db-shared-db-v13.test-multi-db-shared.svc"' -e < /tmp/test-multi-db-shared-json
jq -r '.database.sslMode == "disable"' -e < /tmp/test-multi-db-shared-json
kubectl get secret --namespace=test-multi-db-shared app-a -o json > /tmp/test-multi-db-shared-a
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-db-shared-a | base64 -d > /tmp/test-multi-db-shared-a-json
jq -r '.database.hostname == "test-multi-db-shared-db-v12.test-multi-db-shared.svc"' -e < /tmp/test-multi-db-shared-a-json
jq -r '.database.sslMode == "disable"' -e < /tmp/test-multi-db-shared-a-json