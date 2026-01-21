#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-minio-app" "test-minio-app"

# Test commands from original yaml file
for i in {1..10}; do kubectl get secret --namespace=test-minio-app puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-minio-app puptoo -o json > /tmp/test-minio-app
jq -r '.data["cdappconfig.json"]' < /tmp/test-minio-app | base64 -d > /tmp/test-minio-app-json
jq -r '.objectStore.buckets[] | select(.requestedName == "first-bucket") | .name == "first-bucket"' -e < /tmp/test-minio-app-json
jq -r '.objectStore.buckets[] | select(.requestedName == "second-bucket") | .name == "second-bucket"' -e < /tmp/test-minio-app-json
jq -r '.objectStore.hostname == "test-minio-app-minio.test-minio-app.svc"' -e < /tmp/test-minio-app-json
jq -r '.objectStore.port == 9000' -e < /tmp/test-minio-app-json
jq -r '.objectStore.accessKey != ""' -e < /tmp/test-minio-app-json
jq -r '.objectStore.secretKey != ""' -e < /tmp/test-minio-app-json