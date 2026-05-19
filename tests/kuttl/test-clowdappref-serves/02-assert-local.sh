#!/bin/bash
# Verify inventory uses local rbac ClowdApp
CONFIG=$(kubectl get secret inventory -n test-serves -o jsonpath='{.data.cdappconfig\.json}' | base64 -d)
HOSTNAME=$(echo "$CONFIG" | jq -r '.endpoints[] | select(.app=="rbac") | .hostname')

# Should use local service hostname
if [[ "$HOSTNAME" == *".test-serves.svc"* ]]; then
  echo "✓ Correctly using local ClowdApp: $HOSTNAME"
  exit 0
else
  echo "✗ Expected local hostname (.test-serves.svc), got: $HOSTNAME"
  exit 1
fi
