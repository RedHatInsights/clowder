#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowdapp-watcher-kafka-strimzi"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowdapp-watcher-kafka-strimzi"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
sleep 5
sh kafka_secret_check.sh 30
