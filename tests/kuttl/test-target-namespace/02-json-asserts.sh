#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-target-namespace"

set -x

# Test commands from original yaml file
for i in {1..15}; do kubectl get clowdenvironment test-target-namespace && break || sleep 1; done; echo "ClowdEnvironment not found"; exit 1
kubectl get clowdenvironment test-target-namespace -o json | jq -r '.status.targetNamespace != ""' -e
