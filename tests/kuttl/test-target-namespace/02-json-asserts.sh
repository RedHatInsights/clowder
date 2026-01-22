#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-target-namespace"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-target-namespace"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the ClowdEnvironment
for i in {1..15}; do
  kubectl get clowdenvironment test-target-namespace && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get clowdenvironment test-target-namespace > /dev/null || { echo "ClowdEnvironment not found after retries"; exit 1; }
kubectl get clowdenvironment test-target-namespace -o json | jq -r '.status.targetNamespace != ""' -e
