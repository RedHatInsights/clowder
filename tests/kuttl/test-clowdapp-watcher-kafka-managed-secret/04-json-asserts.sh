#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-managed-secret"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-kafka-managed-secret puptoo -o json > /tmp/test-clowdapp-watcher-kafka-managed-secret2
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-managed-secret2 | base64 -d > /tmp/test-clowdapp-watcher-kafka-managed-secret2-json
jq -r '.kafka.brokers[0].sasl.password == "kafka-new-password"' -e < /tmp/test-clowdapp-watcher-kafka-managed-secret2-json
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-managed-secret-json > /tmp/test-clowdapp-watcher-kafka-managed-secret-hash-cache
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-managed-secret2-json > /tmp/test-clowdapp-watcher-kafka-managed-secret-hash-cache2
diff /tmp/test-clowdapp-watcher-kafka-managed-secret-hash-cache /tmp/test-clowdapp-watcher-kafka-managed-secret-hash-cache2 > /dev/null || exit 0 && exit 1
