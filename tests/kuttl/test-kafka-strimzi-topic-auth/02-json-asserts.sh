#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-strimzi-topic-auth" "test-kafka-strimzi-topic-auth"

# Test commands from original yaml file
for i in {1..5}; do kubectl get secret --namespace=test-kafka-strimzi-topic-auth puptoo -o json > /tmp/test-kafka-strimzi-topic-auth && jq -r '.data["cdappconfig.json"]' < /tmp/test-kafka-strimzi-topic-auth | base64 -d > /tmp/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-strimzi-topic-auth-kafka-bootstrap.test-kafka-strimzi-topic-auth-kafka.svc"' -e < /tmp/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].port == 9093' -e < /tmp/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.username == "test-kafka-strimzi-topic-auth-puptoo"' -e < /tmp/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < /tmp/test-kafka-strimzi-topic-auth-json && jq -r '.kafka.brokers[0].sasl.saslMechanism == "SCRAM-SHA-512"' -e < /tmp/test-kafka-strimzi-topic-auth-json && break || sleep 1; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1