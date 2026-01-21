#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-tls-web-services" "test-tls-web-services"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-tls-web-services puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-tls-web-services puptoo -o json > /tmp/test-tls-web-services
jq -r '.data["cdappconfig.json"]' < /tmp/test-tls-web-services | base64 -d > /tmp/test-tls-web-services-json
jq -r '.publicPort == 8000' -e < /tmp/test-tls-web-services-json
jq -r '.metricsPort == 9000' -e < /tmp/test-tls-web-services-json
jq -r '.privatePort == 10000' -e < /tmp/test-tls-web-services-json
jq -r '.metricsPath == "/metrics"' -e < /tmp/test-tls-web-services-json
jq -r '.endpoints[0].port == 8000' -e < /tmp/test-tls-web-services-json
jq -r '.privateEndpoints[0].port == 10000' -e < /tmp/test-tls-web-services-json
jq -r '.endpoints[0].tlsPort == 8800' -e < /tmp/test-tls-web-services-json
jq -r '.privateEndpoints[0].tlsPort == 18800' -e < /tmp/test-tls-web-services-json