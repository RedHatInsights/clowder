#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-strimzi"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-kafka-strimzi"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sleep 1
kubectl get secret --namespace=test-clowdapp-watcher-kafka-strimzi puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.password' < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json > ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json-pw
jq -r '.hashCache' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json > ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-hash-cache
jq -r '.kafka.brokers[0].hostname == "test-clowdapp-watcher-kafka-strimzi-kafka-bootstrap.test-clowdapp-watcher-kafka-strimzi-kafka.svc"' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].port == 9093' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.username == "test-clowdapp-watcher-kafka-strimzi-puptoo"' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.saslMechanism == "SCRAM-SHA-512"' -e < ${TMP_DIR}/test-clowdapp-watcher-kafka-strimzi-json
