#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-logging-app-interface-cloudwatch" "test-clowdapp-watcher-logging-app-interface-clowdwatch"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-logging-app-interface-clowdwatch puptoo -o json > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch | base64 -d > /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
jq -r '.logging.cloudwatch.secretAccessKey == "top-secret"' -e < /tmp/test-clowdapp-watcher-logging-app-interface-clowdwatch-json