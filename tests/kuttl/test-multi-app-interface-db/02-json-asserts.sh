#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-multi-app-interface-db" "test-multi-app-interface-db-default-ca"

# Test commands from original yaml file
bash -c 'for i in {1..30}; do kubectl get secret --namespace=test-multi-app-interface-db-default-ca app-default-ca && exit 0 || sleep 1; done; echo "Secret not found"; exit 1'
curl https://truststore.pki.rds.amazonaws.com/us-east-1/us-east-1-bundle.pem > us-east-1-bundle.pem
curl https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem > default-ca-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db-default-ca app-default-ca -o json > /tmp/test-multi-app-interface-db-default-ca
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-app-interface-db-default-ca | base64 -d > /tmp/test-multi-app-interface-db-default-ca-json
jq -r '.database.hostname == "app-default-ca.rds.example.com"' -e < /tmp/test-multi-app-interface-db-default-ca-json
jq -r '.database.sslMode == "verify-full"' -e < /tmp/test-multi-app-interface-db-default-ca-json
jq -r '.database.username == "user"' -e < /tmp/test-multi-app-interface-db-default-ca-json
jq -r '.database.rdsCa' < /tmp/test-multi-app-interface-db-default-ca-json > actual.pem
diff --ignore-blank-lines actual.pem default-ca-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-c -o json > /tmp/test-multi-app-interface-db-c
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-app-interface-db-c | base64 -d > /tmp/test-multi-app-interface-db-json-c
jq -r '.database.hostname == "app-b-stage.rds.example.com"' -e < /tmp/test-multi-app-interface-db-json-c
jq -r '.database.sslMode == "verify-full"' -e < /tmp/test-multi-app-interface-db-json-c
jq -r '.database.username == "user"' -e < /tmp/test-multi-app-interface-db-json-c
jq -r '.database.rdsCa' < /tmp/test-multi-app-interface-db-json-c > actual.pem
diff --ignore-blank-lines actual.pem us-east-1-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-b -o json > /tmp/test-multi-app-interface-db-b
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-app-interface-db-b | base64 -d > /tmp/test-multi-app-interface-db-json-b
jq -r '.database.hostname == "app-b-stage.rds.example.com"' -e < /tmp/test-multi-app-interface-db-json-b
jq -r '.database.sslMode == "verify-full"' -e < /tmp/test-multi-app-interface-db-json-b
jq -r '.database.username == "user"' -e < /tmp/test-multi-app-interface-db-json-b
jq -r '.database.rdsCa' < /tmp/test-multi-app-interface-db-json-c > actual.pem
diff --ignore-blank-lines actual.pem us-east-1-bundle.pem
kubectl get secret --namespace=test-multi-app-interface-db app-d -o json > /tmp/test-multi-app-interface-db-d
jq -r '.data["cdappconfig.json"]' < /tmp/test-multi-app-interface-db-d | base64 -d > /tmp/test-multi-app-interface-db-json-d
jq -r '.database.hostname == "unusual.db.name.example.com"' -e < /tmp/test-multi-app-interface-db-json-d