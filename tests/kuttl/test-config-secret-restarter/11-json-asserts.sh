#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-config-secret-restarter" "test-config-secret-restarter"

# Test commands from original yaml file
sh wait_for_generation.sh puptoo-processor "5"
kubectl get secret --namespace=test-config-secret-restarter puptoo -o json > /tmp/test-config-secret-restarter
jq -r '.data["cdappconfig.json"]' < /tmp/test-config-secret-restarter | base64 -d > /tmp/test-config-secret-restarter-json
jq -r '.hashCache == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < /tmp/test-config-secret-restarter-json