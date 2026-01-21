#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multi-db" "test-multi-db"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-multi-db app-c && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-multi-db app-c -o json > /tmp/test-multi-db
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-db | base64 -d > /tmp/test-multi-db-json
jq -r '.database.hostname == "app-b-db.test-multi-db.svc"' -e < /tmp/test-multi-db-json
jq -r '.database.sslMode == "disable"' -e < /tmp/test-multi-db-json
