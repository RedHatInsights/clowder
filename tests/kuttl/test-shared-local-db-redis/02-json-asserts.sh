#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-shared-local-db-redis" "test-local-db-redis-shared"

# Test commands from original yaml file
bash -c 'for i in {1..30}; do kubectl get secret --namespace=test-local-db-redis-shared app-b && exit 0 || sleep 1; done; echo "Secret not found"; exit 1'
kubectl get secret --namespace=test-local-db-redis-shared app-b -o json > /tmp/test-local-db-redis-shared-json-b
jq -r '.data["cdappconfig.json"]' < /tmp/test-local-db-redis-shared-json-b | base64 -d > /tmp/app-b-cdappconfig-json
jq -r '.inMemoryDb.hostname == "app-a-redis.test-local-db-redis-shared.svc"' -e < /tmp/app-b-cdappconfig-json
jq -r '.inMemoryDb.port == 6379' -e < /tmp/app-b-cdappconfig-json