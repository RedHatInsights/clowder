#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-pullsecrets"

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-pullsecrets puptoo -o json > /tmp/test-clowdapp-watcher-pullsecrets2
jq -r '.data["cdappconfig.json"]' < /tmp/test-clowdapp-watcher-pullsecrets2 | base64 -d > /tmp/test-clowdapp-watcher-pullsecrets2-json
jq -r '.hashCache == "d30ceb80d107ba10ba4c271e60c34ef2ce9f8becdb89983544586cebf4e6acb9e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < /tmp/test-clowdapp-watcher-pullsecrets2-json
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-pullsecrets-json > /tmp/test-clowdapp-watcher-pullsecrets-hash-cache
jq -r '.hashCache' -e < /tmp/test-clowdapp-watcher-pullsecrets2-json > /tmp/test-clowdapp-watcher-pullsecrets-hash-cache2
diff /tmp/test-clowdapp-watcher-pullsecrets-hash-cache /tmp/test-clowdapp-watcher-pullsecrets-hash-cache2 > /dev/null || exit 0 && exit 1
