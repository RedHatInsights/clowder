#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Test configuration
TEST_NAME="test-v2-endpoints-clowdapp-tls"
NAMESPACE="test-v2-clowdapp-tls"
APP_NAME="consumer"

# Setup error handling
setup_error_handling "${TEST_NAME}"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/${TEST_NAME}"
mkdir -p "${TMP_DIR}"

set -x

# Wait for secret to be created
for i in {1..10}; do
    kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" && break || sleep 1
done

# Extract config from secret
kubectl get secret --namespace="${NAMESPACE}" "${APP_NAME}" -o json > "${TMP_DIR}/${TEST_NAME}"
jq -r '.data["cdappconfig.json"]' < "${TMP_DIR}/${TEST_NAME}" | base64 -d > "${TMP_DIR}/${TEST_NAME}-json"

# Run assertions - verify v2 endpoint structure with TLS (in-cluster with CA)
jq -r '.dependencyEndpoints.v2.rbac.service.uri == "https://rbac-service.test-v2-clowdapp-tls.svc:8443"' -e < "${TMP_DIR}/${TEST_NAME}-json"
jq -r '.dependencyEndpoints.v2.rbac.service.ca_certificate == "/cdapp/certs/service-ca.crt"' -e < "${TMP_DIR}/${TEST_NAME}-json"

# Verify authenticated is false for ClowdApp (in-cluster) dependencies
jq -r '.dependencyEndpoints.v2.rbac.service.authenticated == false' -e < "${TMP_DIR}/${TEST_NAME}-json"

# Verify CA volume is mounted in consumer deployment
kubectl get deployment --namespace="${NAMESPACE}" consumer-api -o json > "${TMP_DIR}/${TEST_NAME}-deployment"
jq -r '.spec.template.spec.volumes[] | select(.name == "tls-ca") | .configMap.name == "openshift-service-ca.crt"' -e < "${TMP_DIR}/${TEST_NAME}-deployment"
jq -r '.spec.template.spec.containers[0].volumeMounts[] | select(.name == "tls-ca") | .mountPath == "/cdapp/certs"' -e < "${TMP_DIR}/${TEST_NAME}-deployment"

echo "All assertions passed!"
