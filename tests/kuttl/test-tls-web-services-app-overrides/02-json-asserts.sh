#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-tls-web-services-app-overrides"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-tls-web-services-app-overrides"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-enabled && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-enabled -o json > ${TMP_DIR}/test-tls-enabled-secret
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-tls-enabled-secret | base64 -d > ${TMP_DIR}/test-tls-enabled-json
for i in {1..10}; do kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-disabled && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-tls-web-services-app-overrides clowdapp-tls-disabled -o json > ${TMP_DIR}/test-tls-disabled-secret
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-tls-disabled-secret | base64 -d > ${TMP_DIR}/test-tls-disabled-json
jq -r '.publicPort == 8000' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.privatePort == 10000' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.metricsPort == 9000' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.metricsPath == "/metrics"' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .port == 8000' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .port == 10000' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 8800' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 18800' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r 'has("tlsCAPath") | not' -e < ${TMP_DIR}/test-tls-enabled-json
jq -r '.publicPort == 8000' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privatePort == 10000' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.metricsPort == 9000' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.metricsPath == "/metrics"' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | .tlsPort == 0' < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | .tlsPort == 0' < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | has("tlsCAPath") | not' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-disabled") | has("tlsCAPath") | not' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r 'has("tlsCAPath") | not' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints | length == 2' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints | length == 2' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 8800' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsPort == 18800' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .tlsCAPath == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.endpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < ${TMP_DIR}/test-tls-disabled-json
jq -r '.privateEndpoints[] | select(.name == "processor" and .app == "clowdapp-tls-enabled") | .hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < ${TMP_DIR}/test-tls-disabled-json
kubectl get clowdenvironment test-tls-web-services-app-overrides -o json > ${TMP_DIR}/test-env-status
jq -r '.status.apps | length == 2' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .name' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .name' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].hostname == "clowdapp-tls-enabled-processor.test-tls-web-services-app-overrides.svc"' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].name == "clowdapp-tls-enabled-processor"' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-enabled") | .deployments[0].port == 8000' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].hostname == "clowdapp-tls-disabled-processor.test-tls-web-services-app-overrides.svc"' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].name == "clowdapp-tls-disabled-processor"' -e < ${TMP_DIR}/test-env-status
jq -r '.status.apps[] | select(.name == "clowdapp-tls-disabled") | .deployments[0].port == 8000' -e < ${TMP_DIR}/test-env-status
echo "=== clowdapp-tls-enabled config ===" && cat ${TMP_DIR}/test-tls-enabled-json
echo "=== clowdapp-tls-disabled config ===" && cat ${TMP_DIR}/test-tls-disabled-json
echo "=== ClowdEnvironment status ===" && cat ${TMP_DIR}/test-env-status
