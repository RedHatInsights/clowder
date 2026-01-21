#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ff-local" "test-ff-local"

# Test commands from original yaml file
for i in {1..15}; do kubectl get secret --namespace=test-ff-local puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-ff-local puptoo -o json > /tmp/test-ff-local
jq -r '.data["cdappconfig.json"]' < /tmp/test-ff-local | base64 -d > /tmp/test-ff-local-json
jq -r '.webPort == 8000' -e < /tmp/test-ff-local-json
jq -r '.metricsPort == 9000' -e < /tmp/test-ff-local-json
jq -r '.metricsPath == "/metrics"' -e < /tmp/test-ff-local-json
jq -r '.featureFlags.hostname == "test-ff-local-featureflags.test-ff-local.svc"' -e < /tmp/test-ff-local-json
jq -r '.featureFlags.port == 4242' -e < /tmp/test-ff-local-json
jq -r '.featureFlags.scheme == "http"' -e < /tmp/test-ff-local-json