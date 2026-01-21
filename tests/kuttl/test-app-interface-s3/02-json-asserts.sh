#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-app-interface-s3"

set -x

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-app-interface-s3 puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-app-interface-s3 puptoo -o json > /tmp/test-app-interface-s3
jq -r '.data["cdappconfig.json"]' < /tmp/test-app-interface-s3 | base64 -d > /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .region == "us-east"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .accessKey == "aws_access_key"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .name == "test-app-interface-s3"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .secretKey == "aws_secret_key"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-app-interface-s3") | .requestedName == "test-app-interface-s3"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .name == "test-iam-s3"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .accessKey == "aws_access_key"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3") | .secretKey == "aws_secret_key"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .name == "test-iam-s3-2"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .accessKey == "aws_access_key"' -e < /tmp/test-app-interface-s3-json
jq -r '.objectStore.buckets[] | select(.requestedName == "test-iam-s3-2") | .secretKey == "aws_secret_key"' -e < /tmp/test-app-interface-s3-json
