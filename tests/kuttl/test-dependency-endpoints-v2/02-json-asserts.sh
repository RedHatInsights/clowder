#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-dependency-endpoints-v2"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-dependency-endpoints-v2"
mkdir -p "${TMP_DIR}"

set -x

# Retry finding the secret
for i in {1..10}; do
  kubectl get secret --namespace=test-dependency-endpoints-v2 app-consumer && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-dependency-endpoints-v2 app-consumer > /dev/null || { echo "Secret not found after retries"; exit 1; }

# Extract cdappconfig.json
kubectl get secret --namespace=test-dependency-endpoints-v2 app-consumer -o json > ${TMP_DIR}/app-consumer-secret
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/app-consumer-secret | base64 -d > ${TMP_DIR}/cdappconfig.json

# Debug: Print the full config
echo "=== Full cdappconfig.json ==="
cat ${TMP_DIR}/cdappconfig.json | jq .

# ===== V1 Endpoint Validation =====
echo "=== Testing V1 endpoints[] ==="

# V1 public endpoints should exist
jq -r '.endpoints | length > 0' -e < ${TMP_DIR}/cdappconfig.json

# V1 public endpoint has correct structure
jq -r '.endpoints[0].name == "service"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.endpoints[0].app == "app-provider"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.endpoints[0].hostname == "app-provider-service.test-dependency-endpoints-v2.svc"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.endpoints[0].port == 8000' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.endpoints[0].tlsPort == 8443' -e < ${TMP_DIR}/cdappconfig.json

# V1 private endpoints should exist
jq -r '.privateEndpoints | length > 0' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateEndpoints[0].name == "service"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateEndpoints[0].app == "app-provider"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateEndpoints[0].port == 10000' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateEndpoints[0].tlsPort == 10443' -e < ${TMP_DIR}/cdappconfig.json

# ===== V2 Public Endpoint Validation =====
echo "=== Testing V2 dependencyEndpoints.v2 ==="

# V2 dependencyEndpoints object should exist
jq -r '.dependencyEndpoints.v2 != null' -e < ${TMP_DIR}/cdappconfig.json

# V2 should have app-provider
jq -r '.dependencyEndpoints.v2["app-provider"] != null' -e < ${TMP_DIR}/cdappconfig.json

# V2 should have service endpoint (base HTTP)
jq -r '.dependencyEndpoints.v2["app-provider"]["service"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service"].uri == "http://app-provider-service.test-dependency-endpoints-v2.svc:8000"' -e < ${TMP_DIR}/cdappconfig.json

# Base HTTP endpoint should NOT have ca_certificate
jq -r '.dependencyEndpoints.v2["app-provider"]["service"].ca_certificate == null' -e < ${TMP_DIR}/cdappconfig.json

# V2 should have service_tls endpoint (HTTPS)
jq -r '.dependencyEndpoints.v2["app-provider"]["service_tls"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service_tls"].uri == "https://app-provider-service.test-dependency-endpoints-v2.svc:8443"' -e < ${TMP_DIR}/cdappconfig.json

# HTTPS endpoint SHOULD have ca_certificate
jq -r '.dependencyEndpoints.v2["app-provider"]["service_tls"].ca_certificate != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service_tls"].ca_certificate == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/cdappconfig.json

# V2 should have service_h2c endpoint
jq -r '.dependencyEndpoints.v2["app-provider"]["service_h2c"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service_h2c"].uri == "http://app-provider-service.test-dependency-endpoints-v2.svc:9800"' -e < ${TMP_DIR}/cdappconfig.json

# V2 should have service_h2c_tls endpoint
jq -r '.dependencyEndpoints.v2["app-provider"]["service_h2c_tls"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service_h2c_tls"].uri == "https://app-provider-service.test-dependency-endpoints-v2.svc:9443"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.dependencyEndpoints.v2["app-provider"]["service_h2c_tls"].ca_certificate == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/cdappconfig.json

# ===== V2 Private Endpoint Validation =====
echo "=== Testing V2 privateDependencyEndpoints.v2 ==="

# V2 privateDependencyEndpoints object should exist
jq -r '.privateDependencyEndpoints.v2 != null' -e < ${TMP_DIR}/cdappconfig.json

# V2 private should have app-provider
jq -r '.privateDependencyEndpoints.v2["app-provider"] != null' -e < ${TMP_DIR}/cdappconfig.json

# V2 private should have service endpoint (base HTTP)
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service"].uri == "http://app-provider-service.test-dependency-endpoints-v2.svc:10000"' -e < ${TMP_DIR}/cdappconfig.json

# V2 private should have service_tls endpoint (HTTPS)
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_tls"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_tls"].uri == "https://app-provider-service.test-dependency-endpoints-v2.svc:10443"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_tls"].ca_certificate == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/cdappconfig.json

# V2 private should have service_h2c endpoint
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_h2c"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_h2c"].uri == "http://app-provider-service.test-dependency-endpoints-v2.svc:10800"' -e < ${TMP_DIR}/cdappconfig.json

# V2 private should have service_h2c_tls endpoint
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_h2c_tls"] != null' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_h2c_tls"].uri == "https://app-provider-service.test-dependency-endpoints-v2.svc:10843"' -e < ${TMP_DIR}/cdappconfig.json
jq -r '.privateDependencyEndpoints.v2["app-provider"]["service_h2c_tls"].ca_certificate == "/cdapp/certs/service-ca.crt"' -e < ${TMP_DIR}/cdappconfig.json

echo "=== All V2 endpoint assertions passed! ==="
