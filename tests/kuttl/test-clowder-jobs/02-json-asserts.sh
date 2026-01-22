#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-clowder-jobs"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-clowder-jobs"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
bash -c 'for i in {1..100}; do kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-clowder-jobs -o json | jq -e '\''.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")'\'' && exit 0 || sleep 1; done; echo "Pod was not successfully started"; exit 1'
kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-clowder-jobs -o json > ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs
kubectl logs `jq -r '.items[0].metadata.name' < ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs` -n test-clowder-jobs > ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs-output
grep "Hi" ${TMP_DIR}/test-clowder-jobs-output
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-clowder-jobs -o json > ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs
kubectl logs `jq -r '.items[0].metadata.name' < ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs` -n test-clowder-jobs > ${TMP_DIR}/kuttl/test-clowder-jobs/test-clowder-jobs-output-hello-cji-runner
grep "Hello!" ${TMP_DIR}/test-clowder-jobs-output-hello-cji-runner
