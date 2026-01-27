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
kubectl get secret --namespace=test-clowdapp-watcher-pullsecrets puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-pullsecrets
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-pullsecrets | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-pullsecrets-json
jq -r '.hashCache == "d5bb6253b6957e7360e88da131050c3653a0d9fa1cdeeae5753b269d13006c16e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < ${TMP_DIR}/test-clowdapp-watcher-pullsecrets-json
