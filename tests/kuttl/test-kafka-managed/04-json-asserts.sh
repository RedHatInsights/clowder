#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-managed" "test-kafka-managed"

# Test commands from original yaml file
bash -c 'for i in {1..30}; do kubectl get secret --namespace=test-kafka-managed puptoo -o json > /tmp/test-kafka-managed && jq -r '\''.data["cdappconfig.json"]'\'' < /tmp/test-kafka-managed | base64 -d > /tmp/test-kafka-managed-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"'\'' -e < /tmp/test-kafka-managed-json && jq -r '\''.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"'\'' -e < /tmp/test-kafka-managed-json && exit 0 || sleep 2; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1'
jq -r '.kafka.topics[] | select(.requestedName == "topicOne") | .name == "test-kafka-topicOne"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "test-kafka-topicTwo"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].hostname == "kafka-host-name"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].port == 27015' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.username == "kafka-username"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.password == "kafka-password"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.securityProtocol == "SASL_SSL"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.saslMechanism == "PLAIN"' -e < /tmp/test-kafka-managed-json
