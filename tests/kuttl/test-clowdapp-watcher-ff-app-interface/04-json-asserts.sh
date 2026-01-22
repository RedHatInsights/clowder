#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-ff-app-interface"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-ff-app-interface"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-ff-app-interface puptoo -o json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface2
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface2 | base64 -d > ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface2-json
jq -r '.featureFlags.clientAccessToken == "app-a-stage.rds.example.com"' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface2-json
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface-hash-cache
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface2-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-ff-app-interface/test-clowdapp-watcher-ff-app-interface-hash-cache2
diff ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface-hash-cache ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface-hash-cache2 > /dev/null || exit 0 && exit 1
