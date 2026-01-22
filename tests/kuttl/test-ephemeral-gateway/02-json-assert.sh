#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ephemeral-gateway"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-ephemeral-gateway"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
for i in {1..5}; do kubectl get secret -n test-ephemeral-gateway test-ephemeral-gateway-test-cert && break || sleep 1; done; echo "Secret not found"; exit 1
rm -fr ${TMP_DIR}/test-ephemeral-gateway
mkdir -p ${TMP_DIR}/test-ephemeral-gateway/
kubectl get secret -n test-ephemeral-gateway -o json test-ephemeral-gateway-test-cert  | jq -r '.data["ca.crt"] | @base64d' > ${TMP_DIR}/test-ephemeral-gateway/ca.pem
kubectl get secret -n test-ephemeral-gateway -o json test-ephemeral-gateway-test-cert  | jq -r '.data["tls.crt"] | @base64d' > ${TMP_DIR}/test-ephemeral-gateway/tls.crt
kubectl get secret -n test-ephemeral-gateway -o json test-ephemeral-gateway-test-cert  | jq -r '.data["tls.key"] | @base64d' > ${TMP_DIR}/test-ephemeral-gateway/tls.key
kubectl delete configmap  -n test-ephemeral-gateway test-ephemeral-gateway-cert-ca --ignore-not-found=true
kubectl create configmap --from-file=/tmp/test-ephemeral-gateway/ca.pem -n test-ephemeral-gateway test-ephemeral-gateway-cert-ca
