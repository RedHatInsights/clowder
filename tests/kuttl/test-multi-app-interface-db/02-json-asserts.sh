#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multi-app-interface-db"

# Create test-specific directory
TMP_DIR="/tmp/kuttl/test-multi-app-interface-db"
mkdir -p "${TMP_DIR}"

set -x

# Test commands from original yaml file
# Retry finding the secret
for i in {1..30}; do
  kubectl get secret --namespace=test-multi-app-interface-db-default-ca app-default-ca && break
  sleep 1
done

# Verify it exists, fail if not
kubectl get secret --namespace=test-multi-app-interface-db-default-ca app-default-ca > /dev/null || { echo "Secret not found"; exit 1; }

curl https://truststore.pki.rds.amazonaws.com/us-east-1/us-east-1-bundle.pem > ${TMP_DIR}/us-east-1-bundle.pem
curl https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem > ${TMP_DIR}/default-ca-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db-default-ca app-default-ca -o json > ${TMP_DIR}/test-multi-app-interface-db-default-ca
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-app-interface-db-default-ca | base64 -d > ${TMP_DIR}/test-multi-app-interface-db-default-ca-json
jq -r '.database.hostname == "app-default-ca.rds.example.com"' -e < ${TMP_DIR}/test-multi-app-interface-db-default-ca-json
jq -r '.database.sslMode == "verify-full"' -e < ${TMP_DIR}/test-multi-app-interface-db-default-ca-json
jq -r '.database.username == "user"' -e < ${TMP_DIR}/test-multi-app-interface-db-default-ca-json
jq -r '.database.rdsCa' < ${TMP_DIR}/test-multi-app-interface-db-default-ca-json > ${TMP_DIR}/actual.pem
diff --ignore-blank-lines ${TMP_DIR}/actual.pem ${TMP_DIR}/default-ca-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-c -o json > ${TMP_DIR}/test-multi-app-interface-db-c
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-app-interface-db-c | base64 -d > ${TMP_DIR}/test-multi-app-interface-db-json-c
jq -r '.database.hostname == "app-b-stage.rds.example.com"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-c
jq -r '.database.sslMode == "verify-full"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-c
jq -r '.database.username == "user"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-c
jq -r '.database.rdsCa' < ${TMP_DIR}/test-multi-app-interface-db-json-c > ${TMP_DIR}/actual.pem
diff --ignore-blank-lines ${TMP_DIR}/actual.pem ${TMP_DIR}/us-east-1-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-b -o json > ${TMP_DIR}/test-multi-app-interface-db-b
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-app-interface-db-b | base64 -d > ${TMP_DIR}/test-multi-app-interface-db-json-b
jq -r '.database.hostname == "app-b-stage.rds.example.com"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-b
jq -r '.database.sslMode == "verify-full"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-b
jq -r '.database.username == "user"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-b
jq -r '.database.rdsCa' < ${TMP_DIR}/test-multi-app-interface-db-json-c > ${TMP_DIR}/actual.pem
diff --ignore-blank-lines ${TMP_DIR}/actual.pem ${TMP_DIR}/us-east-1-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-d -o json > ${TMP_DIR}/test-multi-app-interface-db-d
jq -r '.data["cdappconfig.json"]' < ${TMP_DIR}/test-multi-app-interface-db-d | base64 -d > ${TMP_DIR}/test-multi-app-interface-db-json-d
jq -r '.database.hostname == "unusual.db.name.example.com"' -e < ${TMP_DIR}/test-multi-app-interface-db-json-d
