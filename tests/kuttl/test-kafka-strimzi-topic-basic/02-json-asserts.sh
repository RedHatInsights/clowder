#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-strimzi-topic-basic" "test-kafka-strimzi-topic"

# Test commands from original yaml file
for i in {1..5}; do kubectl get secret --namespace=test-kafka-strimzi-topic puptoo -o json > /tmp/test-kafka-strimzi-topic && jq -r '.data["cdappconfig.json"]' < /tmp/test-kafka-strimzi-topic | base64 -d > /tmp/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].hostname == "strimzi-topic-basic-kafka-bootstrap.test-kafka-strimzi-topic-kafka.svc"' -e < /tmp/test-kafka-strimzi-topic-json && jq -r '.kafka.brokers[0].port == 9092' -e < /tmp/test-kafka-strimzi-topic-json && break || sleep 1; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1
