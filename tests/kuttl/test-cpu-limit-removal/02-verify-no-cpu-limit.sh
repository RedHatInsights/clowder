#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-cpu-limit-removal"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-cpu-limit-removal"
mkdir -p "${TMP_DIR}"

set -x
# Verify that the deployment for test-app-no-cpu-limit-processor does NOT have a CPU limit

# Get the CPU limit from the deployment (if it exists)
CPU_LIMIT=$(kubectl get deployment test-app-no-cpu-limit-processor -n test-cpu-no-limit -o jsonpath='{.spec.template.spec.containers[0].resources.limits.cpu}')

if [ -z "$CPU_LIMIT" ]; then
    echo "SUCCESS: No CPU limit is set on the deployment (as expected)"
    exit 0
else
    echo "FAILURE: CPU limit '$CPU_LIMIT' is set on the deployment, but it should NOT be set"
    exit 1
fi
