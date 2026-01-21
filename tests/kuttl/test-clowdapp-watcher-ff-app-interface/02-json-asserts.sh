#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-ff-app-interface" "test-clowdapp-watcher-ff-app-interface"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-ff-app-interface puptoo -o json > /tmp/test-clowdapp-watcher-ff-app-interface
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-ff-app-interface | base64 -d > /tmp/test-clowdapp-watcher-ff-app-interface-json
jq -r '.featureFlags.clientAccessToken == "app-b-stage.rds.example.com"' -e < /tmp/test-clowdapp-watcher-ff-app-interface-json