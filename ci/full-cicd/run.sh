#!/usr/bin/env bash
set -euo pipefail

# Verbosity controls
VERBOSE=${VERBOSE:-0}
PYTEST_ARGS=${PYTEST_ARGS:-}
if [ "$VERBOSE" = "1" ]; then
  set -x
  PYTEST_ARGS=${PYTEST_ARGS:-"-vv -s"}
fi

# Ensure deploy and tests run
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

"$SCRIPT_DIR/deploy.sh"

# Run tests
set +e
pytest $PYTEST_ARGS "$SCRIPT_DIR/tests"
rc=$?
set -e

# Cleanup created resources (keep namespace intact)
NS=${TEST_NS:-clowder-e2e}
if [ -f /tmp/resources.yaml ]; then
  echo "Deleting resources from /tmp/resources.yaml"
  oc delete -f /tmp/resources.yaml -n "$NS" --ignore-not-found=true --wait=true || true
fi
echo "Cleaning up leftover resources in namespace $NS"
oc -n "$NS" delete clowdapp --all --ignore-not-found=true || true
oc -n "$NS" delete all --all --ignore-not-found=true || true
oc -n "$NS" delete pvc --all --ignore-not-found=true || true
oc -n "$NS" delete configmap --all --ignore-not-found=true || true
oc -n "$NS" delete secret --all --ignore-not-found=true || true

exit $rc
