#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-app-autoscale"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-app-autoscale"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the resource and checking kafka config
for i in {1..15}; do
  kubectl get secret --namespace=test-kafka-app-autoscale puptoo -o json > ${TMP_DIR}/test-kafka-app-autoscale && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-app-autoscale | base64 -d > ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topicone") | .name == "topicone"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topictwo") | .name == "topictwo"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].hostname == "test-kafka-app-autoscale-kafka-bootstrap.test-kafka-app-autoscale.svc"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].port == 9092' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-kafka-app-autoscale puptoo -o json > ${TMP_DIR}/test-kafka-app-autoscale && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-app-autoscale | base64 -d > ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topicone") | .name == "topicone"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topictwo") | .name == "topictwo"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].hostname == "test-kafka-app-autoscale-kafka-bootstrap.test-kafka-app-autoscale.svc"' -e < ${TMP_DIR}/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].port == 9092' -e < ${TMP_DIR}/test-kafka-app-autoscale-json > /dev/null || { echo "Expected kafka topics config not found in cdappconfig.json"; exit 1; }
