#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-config-secret-restarter"

set -x

# Test commands from original yaml file
sh wait_for_generation.sh puptoo-processor "3"
kubectl get secret --namespace=test-config-secret-restarter puptoo -o json > /tmp/test-config-secret-restarter
jq -r '.data["cdappconfig.json"]' < /tmp/test-config-secret-restarter | base64 -d > /tmp/test-config-secret-restarter-json
jq -r '.hashCache == "a1c3e654937f58a3b14f8257a51db175fc25e82df820104dea4d9f68d5ba9eabe3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"' -e < /tmp/test-config-secret-restarter-json
