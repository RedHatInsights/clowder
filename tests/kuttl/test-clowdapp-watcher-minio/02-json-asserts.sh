#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-minio"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-minio"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sleep 5
kubectl get secret --namespace=test-clowdapp-watcher-minio puptoo -o json > ${TMP_DIR}/test-clowdapp-watcher-minio
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-clowdapp-watcher-minio | base64 -d > ${TMP_DIR}/test-clowdapp-watcher-minio-json
jq -r '.hashCache != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < ${TMP_DIR}/test-clowdapp-watcher-minio-json
jq -r '.objectStore.secretKey != ""' -e < ${TMP_DIR}/test-clowdapp-watcher-minio-json
