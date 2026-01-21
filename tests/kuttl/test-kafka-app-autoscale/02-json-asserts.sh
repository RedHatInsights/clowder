#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-app-autoscale" "test-kafka-app-autoscale"

# Test commands from original yaml file
for i in {1..15}; do kubectl get secret --namespace=test-kafka-app-autoscale puptoo -o json > /tmp/test-kafka-app-autoscale && jq -r '.data["cdappconfig.json"]' < /tmp/test-kafka-app-autoscale | base64 -d > /tmp/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topicone") | .name == "topicone"' -e < /tmp/test-kafka-app-autoscale-json && jq -r '.kafka.topics[] | select(.requestedName == "topictwo") | .name == "topictwo"' -e < /tmp/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].hostname == "test-kafka-app-autoscale-kafka-bootstrap.test-kafka-app-autoscale.svc"' -e < /tmp/test-kafka-app-autoscale-json && jq -r '.kafka.brokers[].port == 9092' -e < /tmp/test-kafka-app-autoscale-json && break || sleep 1; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1