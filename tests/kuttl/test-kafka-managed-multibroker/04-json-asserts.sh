#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-managed-multibroker"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-kafka-managed-multibroker"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
bash -c 'for i in {1..30}; do kubectl get secret --namespace=test-kafka-managed-multibroker puptoo -o json > ${TMP_DIR}/test-kafka-managed-multibroker && jq -r '\''.data["cdappconfig.json"]'\'' < ${TMP_DIR}/test-kafka-managed-multibroker | base64 -d > ${TMP_DIR}/test-kafka-managed-multibroker-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"'\'' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"'\'' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json && exit 0 || sleep 2; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1'
jq -r '.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers | length == 3' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].hostname == "kafka-host-name-0"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[1].hostname == "kafka-host-name-1"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[2].hostname == "kafka-host-name-2"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].port == 27015' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].sasl.username == "kafka-username"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].sasl.password == "kafka-password"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
jq -r '.kafka.brokers[0].sasl.saslMechanism == "PLAIN"' -e < ${TMP_DIR}/test-kafka-managed-multibroker-json
