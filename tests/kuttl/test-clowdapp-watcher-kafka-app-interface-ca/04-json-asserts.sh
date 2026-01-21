#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-app-interface-ca" "test-clowdapp-watcher-kafka-app-interface-ca"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-kafka-app-interface-ca puptoo -o json > /tmp/test-clowdapp-watcher-kafka-app-interface-ca
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-kafka-app-interface-ca | base64 -d > /tmp/test-clowdapp-watcher-kafka-app-interface-ca2-json
jq -r '.kafka.brokers[0].cacert == "new-cacert"' -e < /tmp/test-clowdapp-watcher-kafka-app-interface-ca2-json
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-app-interface-ca-json > /tmp/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-kafka-app-interface-ca2-json > /tmp/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache2
diff /tmp/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache /tmp/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache2 > /dev/null || exit 0 && exit 1