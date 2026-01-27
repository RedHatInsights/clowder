#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-msk"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-kafka-msk"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sleep 5
kubectl get secret --namespace=test-clowdapp-watcher-kafka-msk test-clowdapp-watcher-kafka-msk-connect2 -o json > ${TMP_DIR}/test-clowdapp-watcher-kafka-msk-user
kubectl get secret --namespace=test-clowdapp-watcher-kafka-msk test-clowdapp-watcher-kafka-msk-cluster-ca-cert -o json > ${TMP_DIR}/test-clowdapp-watcher-kafka-msk-cluster-ca-cert
sh create_json.sh
sh create_certs.sh
kubectl apply -f ${TMP_DIR}/watcher-managed-secret.yaml -n test-clowdapp-watcher-kafka-msk-sec-source
kubectl apply -f ${TMP_DIR}/test-clowdapp-watcher-kafka-msk-ca-cert.yaml -n test-clowdapp-watcher-kafka-msk-sec-source
kubectl apply -f ${TMP_DIR}/test-clowdapp-watcher-kafka-msk-connect-user.yaml -n test-clowdapp-watcher-kafka-msk-sec-source
