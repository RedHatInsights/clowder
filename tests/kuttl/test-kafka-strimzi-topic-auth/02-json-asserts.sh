#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-strimzi-topic-auth"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-strimzi-topic-auth"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the resource and checking kafka config
for i in {1..5}; do
  kubectl get secret --namespace=test-kafka-strimzi-topic-auth puptoo -o json > ${TMP_DIR}/test-kafka-strimzi-topic-auth && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-strimzi-topic-auth | base64 -d > ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-strimzi-topic-auth-kafka-bootstrap.test-kafka-strimzi-topic-auth-kafka.svc"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].port == 9093' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.username == "test-kafka-strimzi-topic-auth-puptoo"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.saslMechanism == "SCRAM-SHA-512"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-kafka-strimzi-topic-auth puptoo -o json > ${TMP_DIR}/test-kafka-strimzi-topic-auth && jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-kafka-strimzi-topic-auth | base64 -d > ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-strimzi-topic-auth-kafka-bootstrap.test-kafka-strimzi-topic-auth-kafka.svc"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].port == 9093' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.username == "test-kafka-strimzi-topic-auth-puptoo"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.saslMechanism == "SCRAM-SHA-512"' -e < ${TMP_DIR}/test-kafka-strimzi-topic-auth-json > /dev/null || { echo "Expected kafka topics config not found in cdappconfig.json"; exit 1; }
