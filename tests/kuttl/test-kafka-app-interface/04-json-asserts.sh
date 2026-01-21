#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-kafka-app-interface" "test-kafka-app-interface"

# Test commands from original yaml file
for i in {1..15}; do kubectl get secret --namespace=test-kafka-app-interface puptoo -o json > /tmp/test-kafka-app-interface && jq -r '.data["cdappconfig.json"]' < /tmp/test-kafka-app-interface | base64 -d > /tmp/test-kafka-app-interface-json && jq -r '.kafka.brokers[0].hostname == "test-kafka-app-interface-kafka-bootstrap.test-kafka-app-interface.svc"' -e < /tmp/test-kafka-app-interface-json && jq -r '(.kafka.brokers[0].port == 9093) or (.kafka.brokers[0].port == 9092)' -e < /tmp/test-kafka-app-interface-json && jq -r 'if .kafka.brokers[0] | has("cacert") then (.kafka.brokers[0].cacert == "cacert") else true end' -e < /tmp/test-kafka-app-interface-json && break || sleep 1; done; echo "Expected kafka topics config not found in cdappconfig.json"; exit 1
