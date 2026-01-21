#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-annotations-job" "test-annotations-job"

# Test commands from original yaml file
bash -c 'for i in {1..100}; do kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-annotations-job -o json | jq -e '\''.items[] | select(.status.phase != "Pending" and .status.phase != "Unknown")'\'' && exit 0 || sleep 1; done; echo "Pod was not successfully started"; exit 1'
kubectl get pod -l app=puptoo -l pod=puptoo-standard-cron -n test-annotations-job -o json > /tmp/test-annotations-job
kubectl logs `jq -r '.items[0].metadata.name' < /tmp/test-annotations-job` -n test-annotations-job > /tmp/test-annotations-job-output
grep "Hi" /tmp/test-annotations-job-output
kubectl get pod -l app=puptoo -l job=puptoo-hello-cji -n test-annotations-job -o json > /tmp/test-annotations-job
kubectl logs `jq -r '.items[0].metadata.name' < /tmp/test-annotations-job` -n test-annotations-job > /tmp/test-annotations-job-output-hello-cji-runner
grep "Hello!" /tmp/test-annotations-job-output-hello-cji-runner
