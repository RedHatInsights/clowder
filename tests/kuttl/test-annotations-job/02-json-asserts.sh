#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-annotations-job"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-annotations-job"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the pod
for i in {1..100}; do
  kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-annotations-job -o json | jq -e '.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")' && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-annotations-job -o json | jq -e '.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")' > /dev/null || { echo "Pod was not successfully started"; exit 1; }

kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-annotations-job -o json > ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job
kubectl logs `jq -r '.items[0].metadata.name' < ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job` -n test-annotations-job > ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job-output
grep "Hi" ${TMP_DIR}/test-annotations-job-output
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-annotations-job -o json > ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job
kubectl logs `jq -r '.items[0].metadata.name' < ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job` -n test-annotations-job > ${TMP_DIR}/kuttl/test-annotations-job/test-annotations-job-output-hello-cji-runner
grep "Hello!" ${TMP_DIR}/test-annotations-job-output-hello-cji-runner
