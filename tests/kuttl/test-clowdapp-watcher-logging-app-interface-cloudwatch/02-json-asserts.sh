#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-logging-app-interface-cloudwatch"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-logging-app-interface-cloudwatch"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-logging-app-interface-clowdwatch puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
jq -r '.logging.cloudwatch.secretAccessKey == "top-secret"' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
jq -r '.logging.type == "cloudwatch"' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
