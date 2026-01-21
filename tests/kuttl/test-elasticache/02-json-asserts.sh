#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-elasticache" "test-elasticache"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-elasticache puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-elasticache puptoo -o json > /tmp/test-elasticache
jq -r '.data["cdappconfig.json"]' < /tmp/test-elasticache | base64 -d > /tmp/test-elasticache-json
jq -r '.inMemoryDb.hostname == "lovely"' -e < /tmp/test-elasticache-json
jq -r '.inMemoryDb.port == 6767' -e < /tmp/test-elasticache-json