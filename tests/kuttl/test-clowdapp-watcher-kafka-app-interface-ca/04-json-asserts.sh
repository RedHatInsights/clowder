#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-app-interface-ca"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-kafka-app-interface-ca"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
kubectl get secret --namespace=test-clowdapp-watcher-kafka-app-interface-ca puptoo -o json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca | base64 -d > ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca2-json
jq -r '.kafka.brokers[0].cacert == "new-cacert"' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca2-json
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache
jq -r '.hashCache' -e < ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca2-json > ${TMP_DIR}/kuttl/test-clowdapp-watcher-kafka-app-interface-ca/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache2
diff ${TMP_DIR}/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache ${TMP_DIR}/test-clowdapp-watcher-kafka-app-interface-ca-hash-cache2 > /dev/null || exit 0 && exit 1
