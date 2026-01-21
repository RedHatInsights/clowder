#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-tls-web-services-app-overrides"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-enabled && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-enabled -o json > /tmp/test-tls-enabled-secret
jq -r '.data["cdappconfig.json"]' < /tmp/test-tls-enabled-secret | base64 -d > /tmp/test-tls-enabled-json
for i in {1..10}; do kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-disabled && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-disabled -o json > /tmp/test-tls-disabled-secret
jq -r '.data["cdappconfig.json"]' < /tmp/test-tls-disabled-secret | base64 -d > /tmp/test-tls-disabled-json
jq -r '.publicPort == 8000' -e < /tmp/test-tls-enabled-json
jq -r '.privatePort == 10000' -e < /tmp/test-tls-enabled-json
jq -r '.metricsPort == 9000' -e < /tmp/test-tls-enabled-json
jq -r '.metricsPath == "/metrics"' -e < /tmp/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .port == 8000' -e < /tmp/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .port == 10000' -e < /tmp/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 8800' -e < /tmp/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 18800' -e < /tmp/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < /tmp/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < /tmp/test-tls-enabled-json
jq -r 'has("tlsCAPath") | not' -e < /tmp/test-tls-enabled-json
jq -r '.publicPort == 8000' -e < /tmp/test-tls-disabled-json
jq -r '.privatePort == 10000' -e < /tmp/test-tls-disabled-json
jq -r '.metricsPort == 9000' -e < /tmp/test-tls-disabled-json
jq -r '.metricsPath == "/metrics"' -e < /tmp/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | .tlsPort == 0' < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | .tlsPort == 0' < /tmp/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | has("tlsCAPath") | not' -e < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | has("tlsCAPath") | not' -e < /tmp/test-tls-disabled-json
jq -r 'has("tlsCAPath") | not' -e < /tmp/test-tls-disabled-json
jq -r '.endpoints | length == 2' -e < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints | length == 2' -e < /tmp/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 8800' -e < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 18800' -e < /tmp/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < /tmp/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < /tmp/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < /tmp/test-tls-disabled-json
kubectl get clowdenvironment test-tls-web-services-app-overrides -o json > /tmp/test-env-status
jq -r '.status.apps | length == 2' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .name' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .name' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].name == "clowdapp-tls-enabled-processor"' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].port == 8000' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].hostname == "clowdapp-tls-disabled-processor.test-tls-web-services-app-overrides.svc"' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].name == "clowdapp-tls-disabled-processor"' -e < /tmp/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].port == 8000' -e < /tmp/test-env-status
echo "=== clowdapp-tls-enabled config ===" && cat /tmp/test-tls-enabled-json
echo "=== clowdapp-tls-disabled config ===" && cat /tmp/test-tls-disabled-json
echo "=== ClowdEnvironment status ===" && cat /tmp/test-env-status
