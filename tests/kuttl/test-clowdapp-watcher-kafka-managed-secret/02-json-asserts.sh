#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-managed-secret"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-kafka-managed-secret puptoo -o json > /tmp/test-clowdapp-watcher-kafka-managed-secret
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-managed-secret | base64 -d > /tmp/test-clowdapp-watcher-kafka-managed-secret-json
jq -r '.kafka.brokers[0].sasl.password == "kafka-password"'  -e < /tmp/test-clowdapp-watcher-kafka-managed-secret-json
