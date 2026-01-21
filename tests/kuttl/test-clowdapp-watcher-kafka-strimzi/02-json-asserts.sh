#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-strimzi" "test-clowdapp-watcher-kafka-strimzi"

# Test commands from original yaml file
sleep 1
kubectl get secret --namespace=test-clowdapp-watcher-kafka-strimzi puptoo -o json > /tmp/test-clowdapp-watcher-kafka-strimzi
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-strimzi | base64 -d > /tmp/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.password' < /tmp/test-clowdapp-watcher-kafka-strimzi-json > /tmp/test-clowdapp-watcher-kafka-strimzi-json-pw
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json > /tmp/test-clowdapp-watcher-kafka-strimzi-hash-cache
jq -r '.kafka.brokers[0].hostname == "test-clowdapp-watcher-kafka-strimzi-kafka-bootstrap.test-clowdapp-watcher-kafka-strimzi-kafka.svc"' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].port == 9093' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.username == "test-clowdapp-watcher-kafka-strimzi-puptoo"' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.securityProtocol == "SASL_SSL"' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json
jq -r '.kafka.brokers[0].sasl.saslMechanism == "SCRAM-SHA-512"' -e < /tmp/test-clowdapp-watcher-kafka-strimzi-json
