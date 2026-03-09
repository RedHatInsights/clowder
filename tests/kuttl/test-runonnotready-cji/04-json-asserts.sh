#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-runonnotready-cji"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-runonnotready-cji"
mkdir -p "${TMP_DIR}"

set -x

# Wait for a pod from the v2 CJI (with updated image) to run
for i in {1..100}; do
  kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | \
    jq -e '.items[] | select(.spec.containers[0].image == "busybox:1.37") | select(.status.phase != "Pending" and .status.phase != "Unknown")' && break
  sleep 1
done

# Verify the pod with the updated image exists and ran
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | \
  jq -e '.items[] | select(.spec.containers[0].image == "busybox:1.37") | select(.status.phase != "Pending" and .status.phase != "Unknown")' > /dev/null || \
  { echo "Pod with updated image busybox:1.37 was not successfully started"; exit 1; }

# Get the pod name and verify the output matches the updated command
POD_NAME=$(kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | \
  jq -r '[.items[] | select(.spec.containers[0].image == "busybox:1.37")][0].metadata.name')

kubectl logs "${POD_NAME}" -n test-runonnotready-jobs > "${TMP_DIR}/test-runonnotready-cji-output-v2"
grep "Hello v2!" "${TMP_DIR}/test-runonnotready-cji-output-v2"
