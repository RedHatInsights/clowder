#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-local-db-redis"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-local-db-redis app-a && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-local-db-redis app-a -o json > /tmp/test-local-db-redis-json-a
jq -r '.data["cdappconfig.json"]' < /tmp/test-local-db-redis-json-a | base64 -d > /tmp/app-a-cdappconfig-json
jq -r '.inMemoryDb.hostname == "app-a-redis.test-local-db-redis.svc"' -e < /tmp/app-a-cdappconfig-json
jq -r '.inMemoryDb.port == 6379' -e < /tmp/app-a-cdappconfig-json
jq '.inMemoryDb | has("username")' < /tmp/app-a-cdappconfig-json | grep -q false
jq '.inMemoryDb | has("password")' < /tmp/app-a-cdappconfig-json | grep -q false
