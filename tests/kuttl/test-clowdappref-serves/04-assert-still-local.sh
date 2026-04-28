#!/bin/bash
# Verify inventory STILL uses local rbac ClowdApp (not remote)
CONFIG=$(kubectl get secret inventory-config -n test-serves -o jsonpath='{.data.cdappconfig\.json}' | base64 -d)
HOSTNAME=$(echo "$CONFIG" | jq -r '.endpoints[] | select(.app=="rbac") | .hostname')

# Should still use local service hostname (not remote)
if [[ "$HOSTNAME" == *".test-serves.svc"* ]]; then
  echo "✓ Still using local ClowdApp (serves is empty): $HOSTNAME"
  exit 0
elif [[ "$HOSTNAME" == "rbac.remote.example.com" ]]; then
  echo "✗ Using remote ClowdAppRef but shouldn't (serves is empty)"
  exit 1
else
  echo "✗ Unexpected hostname: $HOSTNAME"
  exit 1
fi
