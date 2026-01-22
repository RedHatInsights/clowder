#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-managed"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-managed"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
bash -c 'for i in {1..30}; do kubectl get secret --namespace=test-kafka-managed puptoo -o json > ${TMP_DIR}/test-kafka-managed && jq -r '\''.data["cdappconfig.json"]'\'' < ${TMP_DIR}/test-kafka-managed | base64 -d > ${TMP_DIR}/test-kafka-managed-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"'\'' -e < ${TMP_DIR}/test-kafka-managed-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"'\'' -e < ${TMP_DIR}/test-kafka-managed-json && exit 0 || sleep 2; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1'
jq -r '.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].hostname == "kafka-host-name"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].port == 27015' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.username == "kafka-username"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.password == "kafka-password"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.securityProtocol == "SASL_SSL"' -e < ${TMP_DIR}/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.saslMechanism == "PLAIN"' -e < ${TMP_DIR}/test-kafka-managed-json
