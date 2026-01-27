#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-strimzi-topic-basic"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-strimzi-topic-basic"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the resource and checking kafka config
for i in {1..5}; do
  kubectl get secret --namespace=test-kafka-strimzi-topic puptoo -o json > ${TMP_DIR}/test-kafka-strimzi-topic && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-strimzi-topic | base64 -d > ${TMP_DIR}/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].hostname == "strimzi-topic-basic-kafka-bootstrap.test-kafka-strimzi-topic-kafka.svc"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].port == 9092' -e < ${TMP_DIR}/test-kafka-strimzi-topic-json && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-kafka-strimzi-topic puptoo -o json > ${TMP_DIR}/test-kafka-strimzi-topic && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-strimzi-topic | base64 -d > ${TMP_DIR}/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].hostname == "strimzi-topic-basic-kafka-bootstrap.test-kafka-strimzi-topic-kafka.svc"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].port == 9092' -e < ${TMP_DIR}/test-kafka-strimzi-topic-json > /dev/null || { echo "Expected kafka topics config not found in cdappconfig.json"; exit 1; }
