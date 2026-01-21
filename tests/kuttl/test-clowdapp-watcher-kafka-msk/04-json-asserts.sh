#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-msk"

# Test commands from original yaml file
sleep 5
kubectl get secret --namespace=test-clowdapp-watcher-kafka-msk-env puptoo -o json > /tmp/test-clowdapp-watcher-kafka-msk-env
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-msk-env | base64 -d > /tmp/test-clowdapp-watcher-kafka-msk-env-json
jq -r '.kafka.brokers[0].sasl.username' < /tmp/test-clowdapp-watcher-kafka-msk-env-json > /tmp/test-clowdapp-watcher-kafka-msk-env-json-user
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-msk-env-json > /tmp/test-clowdapp-watcher-kafka-msk-env-hash-cache
jq -r '.kafka.brokers[0].hostname == "test-clowdapp-watcher-kafka-msk-kafka-bootstrap.test-clowdapp-watcher-kafka-msk.svc"' -e < /tmp/test-clowdapp-watcher-kafka-msk-env-json
jq -r '.kafka.brokers[0].sasl.username == "test-clowdapp-watcher-kafka-msk-connect"' -e < /tmp/test-clowdapp-watcher-kafka-msk-env-json
