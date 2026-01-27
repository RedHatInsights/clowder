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
kubectl get secret --namespace=test-clowdapp-watcher-ff-app-interface puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface-json
jq -r '.featureFlags.clientAccessToken == "app-b-stage.rds.example.com"' -e < ${TMP_DIR}/test-clowdapp-watcher-ff-app-interface-json
