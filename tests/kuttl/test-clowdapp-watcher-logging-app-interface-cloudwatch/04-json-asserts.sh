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
kubectl get secret --namespace=test-clowdapp-watcher-logging-app-interface-clowdwatch puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2 | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
cat ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-json
cat ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
jq -r '.logging.cloudwatch.secretAccessKey == "strong-top-secret"' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
jq -r '.logging.type == "cloudwatch"' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json
jq -r '.hashCache' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-json > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache
jq -r '.hashCache' -e < ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch2-json > ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache2
diff ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache ${TMP_DIR}/test-clowdapp-watcher-logging-app-interface-clowdwatch-hash-cache2 > /dev/null || exit 0 && exit 1
