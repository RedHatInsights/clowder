#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-app-interface"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-app-interface"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the resource and checking kafka config
for i in {1..15}; do
  kubectl get secret --namespace=test-kafka-app-interface puptoo -o json > ${TMP_DIR}/test-kafka-app-interface && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-app-interface | base64 -d > ${TMP_DIR}/test-kafka-app-interface-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-app-interface-kafka-bootstrap.test-kafka-app-interface.svc"' -e < ${TMP_DIR}/test-kafka-app-interface-json && jq -r '.kafka.brokers[0].port == 9092' -e < ${TMP_DIR}/test-kafka-app-interface-json && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-kafka-app-interface puptoo -o json > ${TMP_DIR}/test-kafka-app-interface && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-app-interface | base64 -d > ${TMP_DIR}/test-kafka-app-interface-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-app-interface-kafka-bootstrap.test-kafka-app-interface.svc"' -e < ${TMP_DIR}/test-kafka-app-interface-json && jq -r '.kafka.brokers[0].port == 9092' -e < ${TMP_DIR}/test-kafka-app-interface-json > /dev/null || { echo "Expected kafka topics config not found in cdappconfig.json"; exit 1; }
