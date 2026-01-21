#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-service-mesh"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-service-mesh puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-service-mesh puptoo -o json > /tmp/test-service-mesh
jq -r '.data["cdappconfig.json"]' < /tmp/test-service-mesh | base64 -d > /tmp/test-service-mesh-json
jq -r '.webPort == 8000' -e < /tmp/test-service-mesh-json
jq -r '.metricsPort == 9000' -e < /tmp/test-service-mesh-json
jq -r '.metricsPath == "/metrics"' -e < /tmp/test-service-mesh-json
