#!/bin/bash

export EPHEM_BASE64=`kubectl get secret test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq '.data | map_values(@base64d)' | jq -r -j '"\(.defaultUsername):\(.defaultPassword)" | @base64'`
export PODNAME=`kubectl get pod -l env-app=test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq -r '.items[0].metadata.name'`
kubectl exec -n test-ephemeral-gateway $PODNAME  -- /bin/bash -c "curl -o /tmp/test-ephemeral-gateway-output -v -H \"Authorization: Basic $EPHEM_BASE64\" puptoo-processor.test-ephemeral-gateway.svc:8080"
kubectl cp -n test-ephemeral-gateway $PODNAME:/tmp/test-ephemeral-gateway-output /tmp/test-ephemeral-gateway-output
grep "./clowder-hello" /tmp/test-ephemeral-gateway-output
