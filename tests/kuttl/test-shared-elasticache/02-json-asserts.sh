#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-shared-elasticache" "test-shared-elasticache-ns2"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-shared-elasticache-ns2 another-app && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-shared-elasticache-ns2 another-app -o json > /tmp/test-shared-elasticache
jq -r '.data["cdappconfig.json"]' < /tmp/test-shared-elasticache | base64 -d > /tmp/test-shared-elasticache-json
jq -r '.inMemoryDb.hostname == "lovely"' -e < /tmp/test-shared-elasticache-json
jq -r '.inMemoryDb.port == 6767' -e < /tmp/test-shared-elasticache-json