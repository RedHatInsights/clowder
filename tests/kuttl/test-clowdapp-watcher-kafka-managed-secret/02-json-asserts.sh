#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-managed-secret"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-kafka-managed-secret"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-kafka-managed-secret puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-kafka-managed-secret
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-kafka-managed-secret | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-kafka-managed-secret-json
jq -r '.kafka.brokers[0].sasl.password == "kafka-password"'  -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-managed-secret-json
