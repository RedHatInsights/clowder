#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdappref"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdappref"
mkdir -p ${TMP_DIR}

set -x

echo "🔍 Testing consumer-app cdappconfig.json structure and content..."

# Get the consumer-app config
kubectl get secret consumer-app -n test-clowdappref -o jsonpath='{.data.cdappconfig\.json}' | base64 -d > ${TMP_DIR}/consumer-app-config.json

echo "📋 Consumer app config:"
cat ${TMP_DIR}/consumer-app-config.json | jq .

# Verify JSON is valid
jq empty ${TMP_DIR}/consumer-app-config.json || (echo "❌ Invalid JSON structure" && exit 1)
echo "✅ Valid JSON structure"

# Check basic structure exists
jq -e '.endpoints' ${TMP_DIR}/consumer-app-config.json > /dev/null || (echo "❌ Missing endpoints section" && exit 1)
echo "✅ Endpoints section exists"

# Check that remote-app-tls endpoints are present with correct structure
AUTH_TLS_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-tls")' ${TMP_DIR}/consumer-app-config.json)
PAYMENT_TLS_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "payment-service" and .app == "remote-app-tls")' ${TMP_DIR}/consumer-app-config.json)

if [ -z "$AUTH_TLS_ENDPOINT" ] || [ "$AUTH_TLS_ENDPOINT" = "null" ]; then
    echo "❌ auth-service endpoint (app: remote-app-tls) not found"
    exit 1
fi
echo "✅ auth-service endpoint (app: remote-app-tls) found"

if [ -z "$PAYMENT_TLS_ENDPOINT" ] || [ "$PAYMENT_TLS_ENDPOINT" = "null" ]; then
    echo "❌ payment-service endpoint (app: remote-app-tls) not found"
    exit 1
fi
echo "✅ payment-service endpoint (app: remote-app-tls) found"

# Check that remote-app-no-tls endpoints are present
AUTH_NO_TLS_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-no-tls")' ${TMP_DIR}/consumer-app-config.json)
PAYMENT_NO_TLS_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "payment-service" and .app == "remote-app-no-tls")' ${TMP_DIR}/consumer-app-config.json)

if [ -z "$AUTH_NO_TLS_ENDPOINT" ] || [ "$AUTH_NO_TLS_ENDPOINT" = "null" ]; then
    echo "❌ auth-service endpoint (app: remote-app-no-tls) not found"
    exit 1
fi
echo "✅ auth-service endpoint (app: remote-app-no-tls) found"

if [ -z "$PAYMENT_NO_TLS_ENDPOINT" ] || [ "$PAYMENT_NO_TLS_ENDPOINT" = "null" ]; then
    echo "❌ payment-service endpoint (app: remote-app-no-tls) not found"
    exit 1
fi
echo "✅ payment-service endpoint (app: remote-app-no-tls) found"

# Verify auth-service configuration for remote-app-tls
AUTH_TLS_HOSTNAME=$(echo "$AUTH_TLS_ENDPOINT" | jq -r '.hostname')
AUTH_TLS_PORT=$(echo "$AUTH_TLS_ENDPOINT" | jq -r '.port')
AUTH_TLS_TLS_PORT=$(echo "$AUTH_TLS_ENDPOINT" | jq -r '.tlsPort')

if [ "$AUTH_TLS_HOSTNAME" != "auth.remote-cluster.example.com" ]; then
    echo "❌ auth-service (remote-app-tls) hostname incorrect: expected 'auth.remote-cluster.example.com', got '$AUTH_TLS_HOSTNAME'"
    exit 1
fi
echo "✅ auth-service (remote-app-tls) hostname correct: $AUTH_TLS_HOSTNAME"

if [ "$AUTH_TLS_PORT" != "8080" ]; then
    echo "❌ auth-service (remote-app-tls) port incorrect: expected '8080', got '$AUTH_TLS_PORT'"
    exit 1
fi
echo "✅ auth-service (remote-app-tls) port correct: $AUTH_TLS_PORT"

# auth-service does not have TLS enabled, so tlsPort should be 0
if [ "$AUTH_TLS_TLS_PORT" != "0" ]; then
    echo "❌ auth-service (remote-app-tls) tlsPort incorrect: expected '0', got '$AUTH_TLS_TLS_PORT'"
    exit 1
fi
echo "✅ auth-service (remote-app-tls) tlsPort correct (disabled): $AUTH_TLS_TLS_PORT"

# Verify auth-service API paths for remote-app-tls (should have single path)
AUTH_TLS_API_PATHS=$(echo "$AUTH_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null | wc -l)
if [ "$AUTH_TLS_API_PATHS" -ne 1 ]; then
    echo "❌ auth-service (remote-app-tls) should have exactly 1 API path, found: $AUTH_TLS_API_PATHS"
    echo "API paths found:"
    echo "$AUTH_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null || echo "No API paths found"
    exit 1
fi
echo "✅ auth-service (remote-app-tls) has single API path ($AUTH_TLS_API_PATHS path)"

# Check specific API path for auth-service (remote-app-tls)
echo "$AUTH_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/remote-app-tls-auth-service/")' > /dev/null || (echo "❌ Missing /api/remote-app-tls-auth-service/ path" && exit 1)
echo "✅ auth-service (remote-app-tls) API path is correct"

# Verify auth-service configuration for remote-app-no-tls
AUTH_NO_TLS_HOSTNAME=$(echo "$AUTH_NO_TLS_ENDPOINT" | jq -r '.hostname')
AUTH_NO_TLS_PORT=$(echo "$AUTH_NO_TLS_ENDPOINT" | jq -r '.port')
AUTH_NO_TLS_TLS_PORT=$(echo "$AUTH_NO_TLS_ENDPOINT" | jq -r '.tlsPort')

if [ "$AUTH_NO_TLS_HOSTNAME" != "auth.remote-cluster.example.com" ]; then
    echo "❌ auth-service (remote-app-no-tls) hostname incorrect: expected 'auth.remote-cluster.example.com', got '$AUTH_NO_TLS_HOSTNAME'"
    exit 1
fi
echo "✅ auth-service (remote-app-no-tls) hostname correct: $AUTH_NO_TLS_HOSTNAME"

if [ "$AUTH_NO_TLS_PORT" != "8080" ]; then
    echo "❌ auth-service (remote-app-no-tls) port incorrect: expected '8080', got '$AUTH_NO_TLS_PORT'"
    exit 1
fi
echo "✅ auth-service (remote-app-no-tls) port correct: $AUTH_NO_TLS_PORT"

# TLS port should be 0 for no-tls variant
if [ "$AUTH_NO_TLS_TLS_PORT" != "0" ]; then
    echo "❌ auth-service (remote-app-no-tls) tlsPort incorrect: expected '0', got '$AUTH_NO_TLS_TLS_PORT'"
    exit 1
fi
echo "✅ auth-service (remote-app-no-tls) tlsPort correct (disabled): $AUTH_NO_TLS_TLS_PORT"

# Verify auth-service API paths for remote-app-no-tls (should have single path)
AUTH_NO_TLS_API_PATHS=$(echo "$AUTH_NO_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null | wc -l)
if [ "$AUTH_NO_TLS_API_PATHS" -ne 1 ]; then
    echo "❌ auth-service (remote-app-no-tls) should have exactly 1 API path, found: $AUTH_NO_TLS_API_PATHS"
    echo "API paths found:"
    echo "$AUTH_NO_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null || echo "No API paths found"
    exit 1
fi
echo "✅ auth-service (remote-app-no-tls) has single API path ($AUTH_NO_TLS_API_PATHS path)"

# Check specific API path for auth-service (remote-app-no-tls)
echo "$AUTH_NO_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/remote-app-no-tls-auth-service/")' > /dev/null || (echo "❌ Missing /api/remote-app-no-tls-auth-service/ path" && exit 1)
echo "✅ auth-service (remote-app-no-tls) API path is correct"

# Verify payment-service configuration for remote-app-tls
PAYMENT_TLS_HOSTNAME=$(echo "$PAYMENT_TLS_ENDPOINT" | jq -r '.hostname')
PAYMENT_TLS_PORT=$(echo "$PAYMENT_TLS_ENDPOINT" | jq -r '.port')
PAYMENT_TLS_TLS_PORT=$(echo "$PAYMENT_TLS_ENDPOINT" | jq -r '.tlsPort')

if [ "$PAYMENT_TLS_HOSTNAME" != "payment.remote-cluster.example.com" ]; then
    echo "❌ payment-service (remote-app-tls) hostname incorrect: expected 'payment.remote-cluster.example.com', got '$PAYMENT_TLS_HOSTNAME'"
    exit 1
fi
echo "✅ payment-service (remote-app-tls) hostname correct: $PAYMENT_TLS_HOSTNAME"

if [ "$PAYMENT_TLS_PORT" != "8080" ]; then
    echo "❌ payment-service (remote-app-tls) port incorrect: expected '8080', got '$PAYMENT_TLS_PORT'"
    exit 1
fi
echo "✅ payment-service (remote-app-tls) port correct: $PAYMENT_TLS_PORT"

if [ "$PAYMENT_TLS_TLS_PORT" != "8443" ]; then
    echo "❌ payment-service (remote-app-tls) tlsPort incorrect: expected '8443', got '$PAYMENT_TLS_TLS_PORT'"
    exit 1
fi
echo "✅ payment-service (remote-app-tls) tlsPort correct: $PAYMENT_TLS_TLS_PORT"

# Check payment-service API paths for remote-app-tls (should have multiple paths)
PAYMENT_TLS_API_PATHS=$(echo "$PAYMENT_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null | wc -l)
if [ "$PAYMENT_TLS_API_PATHS" -ne 2 ]; then
    echo "❌ payment-service (remote-app-tls) should have exactly 2 API paths, found: $PAYMENT_TLS_API_PATHS"
    echo "API paths found:"
    echo "$PAYMENT_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null || echo "No API paths found"
    exit 1
fi
echo "✅ payment-service (remote-app-tls) has 2 API paths ($PAYMENT_TLS_API_PATHS paths)"

# Check specific API paths for payment-service (remote-app-tls)
echo "$PAYMENT_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/payment1/")' > /dev/null || (echo "❌ Missing /api/payment1/ path" && exit 1)
echo "$PAYMENT_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/payment2/")' > /dev/null || (echo "❌ Missing /api/payment2/ path" && exit 1)
echo "✅ payment-service (remote-app-tls) API paths are correct"

# Verify payment-service configuration for remote-app-no-tls
PAYMENT_NO_TLS_HOSTNAME=$(echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -r '.hostname')
PAYMENT_NO_TLS_PORT=$(echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -r '.port')
PAYMENT_NO_TLS_TLS_PORT=$(echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -r '.tlsPort')

if [ "$PAYMENT_NO_TLS_HOSTNAME" != "payment.remote-cluster.example.com" ]; then
    echo "❌ payment-service (remote-app-no-tls) hostname incorrect: expected 'payment.remote-cluster.example.com', got '$PAYMENT_NO_TLS_HOSTNAME'"
    exit 1
fi
echo "✅ payment-service (remote-app-no-tls) hostname correct: $PAYMENT_NO_TLS_HOSTNAME"

if [ "$PAYMENT_NO_TLS_PORT" != "8080" ]; then
    echo "❌ payment-service (remote-app-no-tls) port incorrect: expected '8080', got '$PAYMENT_NO_TLS_PORT'"
    exit 1
fi
echo "✅ payment-service (remote-app-no-tls) port correct: $PAYMENT_NO_TLS_PORT"

# TLS port should be 0 for no-tls variant
if [ "$PAYMENT_NO_TLS_TLS_PORT" != "0" ]; then
    echo "❌ payment-service (remote-app-no-tls) tlsPort incorrect: expected '0', got '$PAYMENT_NO_TLS_TLS_PORT'"
    exit 1
fi
echo "✅ payment-service (remote-app-no-tls) tlsPort correct (disabled): $PAYMENT_NO_TLS_TLS_PORT"

# Check payment-service API paths for remote-app-no-tls (should have multiple paths)
PAYMENT_NO_TLS_API_PATHS=$(echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null | wc -l)
if [ "$PAYMENT_NO_TLS_API_PATHS" -ne 2 ]; then
    echo "❌ payment-service (remote-app-no-tls) should have exactly 2 API paths, found: $PAYMENT_NO_TLS_API_PATHS"
    echo "API paths found:"
    echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -r '.apiPaths[]?' 2>/dev/null || echo "No API paths found"
    exit 1
fi
echo "✅ payment-service (remote-app-no-tls) has 2 API paths ($PAYMENT_NO_TLS_API_PATHS paths)"

# Check specific API paths for payment-service (remote-app-no-tls)
echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/payment1/")' > /dev/null || (echo "❌ Missing /api/payment1/ path" && exit 1)
echo "$PAYMENT_NO_TLS_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/payment2/")' > /dev/null || (echo "❌ Missing /api/payment2/ path" && exit 1)
echo "✅ payment-service (remote-app-no-tls) API paths are correct"

    # Check consumer-app-processor endpoint (self-reference)
CONSUMER_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "processor" and .app == "consumer-app")' ${TMP_DIR}/consumer-app-config.json)
if [ -z "$CONSUMER_ENDPOINT" ] || [ "$CONSUMER_ENDPOINT" = "null" ]; then
    echo "❌ processor endpoint (app: consumer-app) not found"
    exit 1
fi
echo "✅ processor endpoint (app: consumer-app) found"

# Check consumer-app-processor API path
echo "$CONSUMER_ENDPOINT" | jq -e '.apiPaths[] | select(. == "/api/consumer-app-processor/")' > /dev/null || (echo "❌ Missing /api/consumer-app-processor/ path" && exit 1)
echo "✅ consumer-app processor API path is correct"

echo "🎉 Consumer app configuration validation complete!"

echo "🔍 Testing mixed-deps-app cdappconfig.json structure and content..."

# Get the mixed-deps-app config
kubectl get secret mixed-deps-app -n test-clowdappref -o jsonpath='{.data.cdappconfig\.json}' | base64 -d > ${TMP_DIR}/mixed-deps-app-config.json

echo "📋 Mixed deps app config:"
cat ${TMP_DIR}/mixed-deps-app-config.json | jq .

# Verify JSON is valid
jq empty ${TMP_DIR}/mixed-deps-app-config.json || (echo "❌ Invalid JSON structure" && exit 1)
echo "✅ Valid JSON structure"

# Count total endpoints (should have ClowdAppRef + ClowdApp endpoints)
TOTAL_ENDPOINTS=$(jq '.endpoints | length' ${TMP_DIR}/mixed-deps-app-config.json)
if [ "$TOTAL_ENDPOINTS" -lt 5 ]; then
    echo "❌ Expected at least 5 endpoints (auth-service from remote-app-tls, payment-service from remote-app-tls, auth-service from remote-app-no-tls, payment-service from remote-app-no-tls, processor from consumer-app), found: $TOTAL_ENDPOINTS"
    jq '.endpoints[] | {name: .name, app: .app}' ${TMP_DIR}/mixed-deps-app-config.json
    exit 1
fi
echo "✅ Found $TOTAL_ENDPOINTS endpoints as expected"

# Check that remote-app-tls endpoints are present (from ClowdAppRef)
jq -e '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-tls")' ${TMP_DIR}/mixed-deps-app-config.json > /dev/null || (echo "❌ auth-service endpoint (app: remote-app-tls) missing" && exit 1)
jq -e '.endpoints[] | select(.name == "payment-service" and .app == "remote-app-tls")' ${TMP_DIR}/mixed-deps-app-config.json > /dev/null || (echo "❌ payment-service endpoint (app: remote-app-tls) missing" && exit 1)
echo "✅ ClowdAppRef remote-app-tls endpoints present"

# Check that remote-app-no-tls endpoints are present (from ClowdAppRef)
jq -e '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-no-tls")' ${TMP_DIR}/mixed-deps-app-config.json > /dev/null || (echo "❌ auth-service endpoint (app: remote-app-no-tls) missing" && exit 1)
jq -e '.endpoints[] | select(.name == "payment-service" and .app == "remote-app-no-tls")' ${TMP_DIR}/mixed-deps-app-config.json > /dev/null || (echo "❌ payment-service endpoint (app: remote-app-no-tls) missing" && exit 1)
echo "✅ ClowdAppRef remote-app-no-tls endpoints present"

# Check that consumer-app endpoints are present (from ClowdApp dependency)
CONSUMER_ENDPOINT=$(jq -r '.endpoints[] | select(.name == "processor" and .app == "consumer-app")' ${TMP_DIR}/mixed-deps-app-config.json)
if [ -z "$CONSUMER_ENDPOINT" ] || [ "$CONSUMER_ENDPOINT" = "null" ]; then
    echo "❌ processor endpoint (app: consumer-app) not found"
    echo "Available endpoints:"
    jq '.endpoints[] | {name: .name, app: .app}' ${TMP_DIR}/mixed-deps-app-config.json
    exit 1
fi
echo "✅ ClowdApp dependency endpoint present"

# Verify consumer-app endpoint points to internal service (not external hostname)
CONSUMER_HOSTNAME=$(echo "$CONSUMER_ENDPOINT" | jq -r '.hostname')
if [[ "$CONSUMER_HOSTNAME" == *".remote-cluster.example.com" ]]; then
    echo "❌ consumer-app endpoint should not have external hostname, got: $CONSUMER_HOSTNAME"
    exit 1
fi
echo "✅ consumer-app endpoint has internal hostname: $CONSUMER_HOSTNAME"

# Verify auth-service still has external hostname (from ClowdAppRef)
AUTH_TLS_HOSTNAME=$(jq -r '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-tls") | .hostname' ${TMP_DIR}/mixed-deps-app-config.json)
if [ "$AUTH_TLS_HOSTNAME" != "auth.remote-cluster.example.com" ]; then
    echo "❌ auth-service (app: remote-app-tls) should have external hostname, got: $AUTH_TLS_HOSTNAME"
    exit 1
fi
echo "✅ auth-service (app: remote-app-tls) has external hostname: $AUTH_TLS_HOSTNAME"

AUTH_NO_TLS_HOSTNAME=$(jq -r '.endpoints[] | select(.name == "auth-service" and .app == "remote-app-no-tls") | .hostname' ${TMP_DIR}/mixed-deps-app-config.json)
if [ "$AUTH_NO_TLS_HOSTNAME" != "auth.remote-cluster.example.com" ]; then
    echo "❌ auth-service (app: remote-app-no-tls) should have external hostname, got: $AUTH_NO_TLS_HOSTNAME"
    exit 1
fi
echo "✅ auth-service (app: remote-app-no-tls) has external hostname: $AUTH_NO_TLS_HOSTNAME"

echo "🎉 Mixed deps app configuration validation complete!"

echo "🔍 Testing endpoint structure and configuration..."

# Check endpoints section in consumer-app
kubectl get secret consumer-app -n test-clowdappref -o jsonpath='{.data.cdappconfig\.json}' | base64 -d > ${TMP_DIR}/consumer-config-deps.json

# Verify endpoints section exists and contains remote-app endpoints
ENDPOINTS_SECTION=$(jq -r '.endpoints // empty' ${TMP_DIR}/consumer-config-deps.json)
if [ -z "$ENDPOINTS_SECTION" ]; then
    echo "❌ Missing endpoints section in consumer-app config"
    exit 1
fi
echo "✅ Endpoints section exists"

# Check if remote-app-tls endpoints are present in endpoints array
jq -e '.endpoints[] | select(.app == "remote-app-tls" and .name == "auth-service")' ${TMP_DIR}/consumer-config-deps.json > /dev/null || (echo "❌ auth-service endpoint (app: remote-app-tls) not found in endpoints" && exit 1)
jq -e '.endpoints[] | select(.app == "remote-app-tls" and .name == "payment-service")' ${TMP_DIR}/consumer-config-deps.json > /dev/null || (echo "❌ payment-service endpoint (app: remote-app-tls) not found in endpoints" && exit 1)
echo "✅ Both auth-service and payment-service endpoints found for remote-app-tls"

# Check if remote-app-no-tls endpoints are present in endpoints array
jq -e '.endpoints[] | select(.app == "remote-app-no-tls" and .name == "auth-service")' ${TMP_DIR}/consumer-config-deps.json > /dev/null || (echo "❌ auth-service endpoint (app: remote-app-no-tls) not found in endpoints" && exit 1)
jq -e '.endpoints[] | select(.app == "remote-app-no-tls" and .name == "payment-service")' ${TMP_DIR}/consumer-config-deps.json > /dev/null || (echo "❌ payment-service endpoint (app: remote-app-no-tls) not found in endpoints" && exit 1)
echo "✅ Both auth-service and payment-service endpoints found for remote-app-no-tls"

# Check if consumer-app processor endpoint is present (self-reference)
jq -e '.endpoints[] | select(.app == "consumer-app" and .name == "processor")' ${TMP_DIR}/consumer-config-deps.json > /dev/null || (echo "❌ processor endpoint (app: consumer-app) not found in endpoints" && exit 1)
echo "✅ processor endpoint found for consumer-app"

# Test mixed-deps-app endpoints
kubectl get secret mixed-deps-app -n test-clowdappref -o jsonpath='{.data.cdappconfig\.json}' | base64 -d > ${TMP_DIR}/mixed-config-deps.json

# Should have both remote-app-tls, remote-app-no-tls, and consumer-app endpoints
jq -e '.endpoints[] | select(.app == "remote-app-tls")' ${TMP_DIR}/mixed-config-deps.json > /dev/null || (echo "❌ remote-app-tls endpoints not found in mixed-deps endpoints" && exit 1)
jq -e '.endpoints[] | select(.app == "remote-app-no-tls")' ${TMP_DIR}/mixed-config-deps.json > /dev/null || (echo "❌ remote-app-no-tls endpoints not found in mixed-deps endpoints" && exit 1)
jq -e '.endpoints[] | select(.app == "consumer-app")' ${TMP_DIR}/mixed-config-deps.json > /dev/null || (echo "❌ consumer-app endpoints not found in mixed-deps endpoints" && exit 1)
echo "✅ remote-app-tls, remote-app-no-tls, and consumer-app endpoints found in mixed-deps"

echo "🎉 Endpoints structure validation complete!"
