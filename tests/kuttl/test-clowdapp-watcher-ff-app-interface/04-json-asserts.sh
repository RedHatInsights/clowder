#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-ff-app-interface" "test-clowdapp-watcher-ff-app-interface"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-ff-app-interface puptoo -o json > /tmp/test-clowdapp-watcher-ff-app-interface2
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-ff-app-interface2 | base64 -d > /tmp/test-clowdapp-watcher-ff-app-interface2-json
jq -r '.featureFlags.clientAccessToken == "app-a-stage.rds.example.com"' -e < /tmp/test-clowdapp-watcher-ff-app-interface2-json
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-ff-app-interface-json > /tmp/test-clowdapp-watcher-ff-app-interface-hash-cache
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-ff-app-interface2-json > /tmp/test-clowdapp-watcher-ff-app-interface-hash-cache2
diff /tmp/test-clowdapp-watcher-ff-app-interface-hash-cache /tmp/test-clowdapp-watcher-ff-app-interface-hash-cache2 > /dev/null || exit 0 && exit 1