#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-runonnotready-cji"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-runonnotready-cji"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
bash -c 'for i in {1..100}; do kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json | jq -e '\''.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")'\'' && exit 0 || sleep 1; done; echo "Pod was not successfully started"; exit 1'
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-runonnotready-jobs -o json > ${TMP_DIR}/test-runonnotready-jobs
kubectl logs `jq -r '.items[0].metadata.name' < ${TMP_DIR}/test-runonnotready-jobs` -n test-runonnotready-jobs > ${TMP_DIR}/test-runonnotready-jobs-output-hello-cji-runner
grep "Hello!" ${TMP_DIR}/test-runonnotready-jobs-output-hello-cji-runner
