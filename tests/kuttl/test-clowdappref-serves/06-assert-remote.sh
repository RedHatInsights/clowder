#!/bin/bash
# Verify inventory NOW uses remote rbac ClowdAppRef
CONFIG=$(kubectl get secret inventory-config -n test-serves -o jsonpath='{.data.cdappconfig\.json}' | base64 -d)
HOSTNAME=$(echo "$CONFIG" | jq -r '.endpoints[] | select(.app=="rbac") | .hostname')

# Should now use remote hostname
if [[ "$HOSTNAME" == "rbac.remote.example.com" ]]; then
  echo "✓ Now using remote ClowdAppRef (inventory in serves): $HOSTNAME"
  exit 0
elif [[ "$HOSTNAME" == *".test-serves.svc"* ]]; then
  echo "✗ Still using local ClowdApp but should use remote (inventory in serves)"
  exit 1
else
  echo "✗ Unexpected hostname: $HOSTNAME"
  exit 1
fi
