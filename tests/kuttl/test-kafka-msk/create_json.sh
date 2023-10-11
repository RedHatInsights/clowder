#!/bin/bash

# Set the file paths
username=$(cat /tmp/test-kafka-msk-user | jq -r '.metadata.name')
password=$(cat /tmp/test-kafka-msk-user | jq -r '.data.password' | base64 -d)
cert=$(cat /tmp/test-kafka-msk-cluster-ca-cert | jq -r '.data["ca.crt"]' | base64 -d)
port=9093
saslMechanism=SCRAM-SHA-512
hostname=test-kafka-msk-kafka-bootstrap.test-kafka-msk.svc

# Create the Kubernetes Secret YAML
cat <<EOF > /tmp/managed-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: managed-secret
type: Opaque
data:
  username: $(echo -n "$username" | base64)
  password: $(echo -n "$password"| base64)
  saslMechanism: $(echo -n "$saslMechanism" | base64)
  port: $(echo -n "$port" | base64)
  hostname: $(echo -n "$hostname" | base64)
  ca.crt: $(echo -n "$cert" | base64 | tr -d '\n')
  cacert: $(echo -n "$cert" | base64 | tr -d '\n')
EOF
