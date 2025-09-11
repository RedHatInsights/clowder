#!/bin/bash

# Set the file paths
cacrt=$(cat /tmp/test-clowdapp-watcher-kafka-msk-cluster-ca-cert | jq -r '.data["ca.crt"]' | base64 -d)
cap12=$(cat /tmp/test-clowdapp-watcher-kafka-msk-cluster-ca-cert | jq -r '.data["ca.p12"]' | base64 -d)
capass=$(cat /tmp/test-clowdapp-watcher-kafka-msk-cluster-ca-cert | jq -r '.data["ca.password"]' | base64 -d)

# Create the Kubernetes Secret YAML
cat <<EOF > /tmp/test-clowdapp-watcher-kafka-msk-ca-cert.yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-clowdapp-watcher-kafka-msk-cluster-ca-cert
type: Opaque
data:
  ca.crt: $(echo -n "$cacrt" | base64 | tr -d '\n')
  ca.p12: $(echo -n "$cap12"| base64 | tr -d '\n')
  password: $(echo -n "$capass" | base64)
EOF

# Set the file paths
password=$(cat /tmp/test-clowdapp-watcher-kafka-msk-user | jq -r '.data["password"]' | base64 -d)
jaas=$(cat /tmp/test-clowdapp-watcher-kafka-msk-user | jq -r '.data["sasl.jaas.config"]' | base64 -d)

# Create the Kubernetes Secret YAML
cat <<EOF > /tmp/test-clowdapp-watcher-kafka-msk-connect-user.yaml
apiVersion: v1
kind: Secret
metadata:
  name: test-clowdapp-watcher-kafka-msk-connect
type: Opaque
data:
  sasl.jaas.config: $(echo -n "$jaas"| base64 | tr -d '\n')
  password: $(echo -n "$password" | base64)
EOF
