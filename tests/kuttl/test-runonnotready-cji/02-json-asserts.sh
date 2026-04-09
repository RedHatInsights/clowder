#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-runonnotready-cji"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-runonnotready-cji"
mkdir -p "${TMP_DIR}"

set -x

# Retry finding the pod
for i in {1..100}; do
  kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | jq -e '.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")' && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | jq -e '.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")' > /dev/null || { echo "Pod was not successfully started"; exit 1; }

# Get pod details and verify the output
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json > "${TMP_DIR}/pod-list.json"
POD_NAME=$(jq -r '.items[0].metadata.name' < "${TMP_DIR}/pod-list.json")
kubectl logs "${POD_NAME}" -n test-runonnotready-jobs > "${TMP_DIR}/output-hello-cji-runner"
grep "Hello!" "${TMP_DIR}/output-hello-cji-runner"
