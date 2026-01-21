#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-logging-app-interface-cloudwatch"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-logging-app-interface-clowdwatch puptoo -o json > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2 | base64 -d > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
cat /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
cat /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
jq -r '.logging.cloudwatch.secretAccessKey == "strong-top-secret"' -e < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-json > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache2
diff /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache2 > /dev/null || exit 0 && exit 1
