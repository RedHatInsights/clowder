#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-msk"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-msk"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-kafka-msk test-kafka-msk-connect && kubectl get secret --namespace=test-kafka-msk test-kafka-msk-cluster-ca-cert && break || sleep 1; done; echo "Expected secrets not found"; exit 1
kubectl get secret --namespace=test-kafka-msk test-kafka-msk-connect -o json > ${TMP_DIR}/test-kafka-msk-user
kubectl get secret --namespace=test-kafka-msk test-kafka-msk-cluster-ca-cert -o json > ${TMP_DIR}/test-kafka-msk-cluster-ca-cert
sh create_json.sh
sh create_certs.sh
kubectl apply -f ${TMP_DIR}/managed-secret.yaml -n test-kafka-msk-sec-source
kubectl apply -f ${TMP_DIR}/test-kafka-msk-ca-cert.yaml -n test-kafka-msk-sec-source
kubectl apply -f ${TMP_DIR}/test-kafka-msk-connect-user.yaml -n test-kafka-msk-sec-source
