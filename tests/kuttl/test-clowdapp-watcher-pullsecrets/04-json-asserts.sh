#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-pullsecrets"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-pullsecrets"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-pullsecrets puptoo -o json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets2
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets2 | base64 -d > ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets2-json
jq -r '.hashCache == "d30ceb80d107ba10ba4c271e60c34ef2ce9f8becdb89983544586cebf4e6acb9e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets2-json
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets-hash-cache
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets2-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-pullsecrets/test-clowdapp-watcher-pullsecrets-hash-cache2
diff ${TMP_DIR}/test-clowdapp-watcher-pullsecrets-hash-cache ${TMP_DIR}/test-clowdapp-watcher-pullsecrets-hash-cache2 > /dev/null || exit 0 && exit 1
