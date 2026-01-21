#!/bin/bash

# Source common error handling
source "$(dirname "$0")/../_common/error-handler.sh"

# Setup error handling
setup_error_handling "test-ephemeral-gateway"

# Test commands from original yaml file
for i in {1..5}; do kubectl get secret --namespace=test-ephemeral-gateway puptoo && break || sleep 1; done; echo "Secret not found"; exit 1
kubectl get secret --namespace=test-ephemeral-gateway puptoo -o json > /tmp/test-ephemeral-gateway-apipaths
jq -r '.data["cdappconfig.json"]' < /tmp/test-ephemeral-gateway-apipaths | base64 -d > /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPath == "/api/puptoo/"' -e < /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.endpoints[] | select(.app == "puptoo") | select(.name == "processor") | .apiPaths[0] == "/api/puptoo/"' -e < /tmp/test-ephemeral-gateway-apipaths-json
kubectl get secret --namespace=test-ephemeral-gateway puptoo-2paths -o json > /tmp/test-ephemeral-gateway-apipaths
jq -r '.data["cdappconfig.json"]' < /tmp/test-ephemeral-gateway-apipaths | base64 -d > /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.endpoints[] | select(.app == "puptoo-2paths") | select(.name == "processor") | .apiPath == "/api/puptoo1/"' -e < /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.endpoints[] | select(.app == "puptoo-2paths") | select(.name == "processor") | .apiPaths[0] == "/api/puptoo1/"' -e < /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.endpoints[] | select(.app == "puptoo-2paths") | select(.name == "processor") | .apiPaths[1] == "/api/puptoo2/"' -e < /tmp/test-ephemeral-gateway-apipaths-json
jq -r '.hostname == "test-ephemeral-gateway"' -e < /tmp/test-ephemeral-gateway-apipaths-json
kubectl get secret test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq -r '.data.defaultUsername == "amRvZQ=="'
kubectl get secret test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq -r '.data.version == "MTUuMC4yCg=="'
kubectl get pod -l pod=puptoo-processor -n test-ephemeral-gateway -o json  | jq -r '.items[0].spec.containers[0].name=="puptoo-processor"'
kubectl get pod -l pod=puptoo-processor -n test-ephemeral-gateway -o json  | jq -r '.items[0].spec.containers[1].name=="crcauth"'
sh test_creds.sh
