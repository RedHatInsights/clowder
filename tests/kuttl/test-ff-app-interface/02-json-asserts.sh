#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ff-app-interface" "test-ff-app-interface"

# Test commands from original yaml file
for i in {1..5}; do kubectl get secret --namespace=test-ff-app-interface puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-ff-app-interface puptoo -o json > /tmp/test-ff-app-interface
jq -r '.data["cdappconfig.json"]' < /tmp/test-ff-app-interface | base64 -d > /tmp/test-ff-app-interface-json
jq -r '.featureFlags.clientAccessToken == "app-b-stage.rds.example.com"' -e < /tmp/test-ff-app-interface-json
jq -r '.featureFlags.hostname == "test.featureflags.redhat.com"' -e < /tmp/test-ff-app-interface-json
jq -r '.featureFlags.port == 12345' -e < /tmp/test-ff-app-interface-json
jq -r '.featureFlags.scheme == "https"' -e < /tmp/test-ff-app-interface-json