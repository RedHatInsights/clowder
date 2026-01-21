#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multiple-app-endpoints" "test-multiple-app-endpoints"

# Test commands from original yaml file
bash -c 'for i in {1..60}; do kubectl get secret --namespace=test-multiple-app-endpoints puptoo-a && exit 0 || sleep 1; done; echo "Secret \"puptoo-a\" not found"; exit 1'
kubectl get secret puptoo-a -o json -n test-multiple-app-endpoints > /tmp/test-multiple-app-endpoints
jq -r '.data["cdappconfig.json"]' < /tmp/test-multiple-app-endpoints | base64 -d > /tmp/test-multiple-app-endpoints-json
bash -c 'for i in {1..60}; do kubectl get secret --namespace=test-multiple-app-endpoints-b puptoo-b && exit 0 || sleep 1; done; echo "Secret \"puptoo-b\" not found"; exit 1'
kubectl get secret puptoo-b -o json -n test-multiple-app-endpoints-b > /tmp/test-multiple-app-endpoints-b
jq -r '.data["cdappconfig.json"]' < /tmp/test-multiple-app-endpoints-b | base64 -d > /tmp/test-multiple-app-endpoints-json-b
jq -r '.endpoints[] | select(.app == "puptoo-a") | .name == "processor"' -e < /tmp/test-multiple-app-endpoints-json
jq -r '.endpoints[] | select(.app == "puptoo-a-2") | .name == "processor"' -e < /tmp/test-multiple-app-endpoints-json
jq -r '.endpoints | length == 2' -e < /tmp/test-multiple-app-endpoints-json
jq -r '.endpoints[] | select(.app == "puptoo-b") | .name == "processor"' -e < /tmp/test-multiple-app-endpoints-json-b
jq -r '.endpoints[] | select(.app == "puptoo-b-2") | .name == "processor"' -e < /tmp/test-multiple-app-endpoints-json-b
jq -r '.endpoints | length == 2' -e < /tmp/test-multiple-app-endpoints-json-b