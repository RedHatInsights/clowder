#!/bin/bash

export EPHEM_BASE64=`kubectl get secret test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq '.data | map_values(@base64d)' | jq -r -j '"\(.defaultUsername):\(.defaultPassword)" | @base64'`
export PODNAME=`kubectl get pod -l env-app=test-ephemeral-gateway-keycloak -n test-ephemeral-gateway -o json | jq -r '.items[0].metadata.name'`
kubectl exec -n test-ephemeral-gateway $PODNAME  -- /bin/bash -c "curl -o /tmp/test-ephemeral-gateway-output -v -H \"Authorization: Basic $EPHEM_BASE64\" puptoo-processor.test-ephemeral-gateway.svc:8080"
kubectl cp -n test-ephemeral-gateway $PODNAME:/tmp/test-ephemeral-gateway-output /tmp/test-ephemeral-gateway-output
grep "./clowder-hello" /tmp/test-ephemeral-gateway-output

echo "OH SNAP"

kubectl exec -n test-ephemeral-gateway $PODNAME -- /bin/bash -c "mkdir -p /tmp/test-ephemeral-gateway"
kubectl cp /tmp/test-ephemeral-gateway/tls.crt test-ephemeral-gateway/$PODNAME:/tmp/test-ephemeral-gateway/tls.crt
kubectl cp /tmp/test-ephemeral-gateway/tls.key test-ephemeral-gateway/$PODNAME:/tmp/test-ephemeral-gateway/tls.key
kubectl cp test.json test-ephemeral-gateway/$PODNAME:/tmp/test-ephemeral-gateway/test.json
kubectl exec -n test-ephemeral-gateway $PODNAME -- /bin/bash -c "curl -k http://test-ephemeral-gateway-mbop:8080/v1/registrations -H \"Authorization: Basic $EPHEM_BASE64\" -vvv -H \"Content-Type: application/json\" -d @/tmp/test-ephemeral-gateway/test.json -H \"x-rh-certauth-cn:/CN=36f23107-9b7c-48f6-8d5b-e6691e7dd235\""
kubectl exec -n test-ephemeral-gateway $PODNAME -- /bin/bash -c "curl -o /tmp/test-ephemeral-gateway-output-2 -v https://test-ephemeral-gateway-cert:9090/api/puptoo/ -vvv --connect-to test-ephemeral-gateway-cert:9090:test-ephemeral-gateway-caddy-gateway:9090 -k --key /tmp/test-ephemeral-gateway/tls.key --cert /tmp/test-ephemeral-gateway/tls.crt"
kubectl cp -n test-ephemeral-gateway $PODNAME:/tmp/test-ephemeral-gateway-output-2 /tmp/test-ephemeral-gateway-output
grep "./clowder-hello" /tmp/test-ephemeral-gateway-output
