#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-managed"

# Test commands from original yaml file
for i in {1..15}; do kubectl get secret --namespace=test-kafka-managed puptoo -o json > /tmp/test-kafka-managed && jq -r '.data["cdappconfig.json"]' < /tmp/test-kafka-managed | base64 -d > /tmp/test-kafka-managed-json && jq -r '.kafka.topics[] | select(.requestedName == "topicOne") | .name == "topicOne"' -e < /tmp/test-kafka-managed-json && jq -r '.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "topicTwo"' -e < /tmp/test-kafka-managed-json && break || sleep 1; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1
jq -r '.kafka.topics[] | select(.requestedName == "topicOne") | .name == "topicOne"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.topics[] | select(.requestedName == "topicTwo") | .name == "topicTwo"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].hostname == "kafka-host-name"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].cacert == "some-pem"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].port == 27015' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.username == "kafka-username"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.password == "kafka-password"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.securityProtocol == "SASL_SSL"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].sasl.saslMechanism == "PLAIN"' -e < /tmp/test-kafka-managed-json
jq -r '.kafka.brokers[].securityProtocol == "SASL_SSL"' -e < /tmp/test-kafka-managed-json
