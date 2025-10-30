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

# Cleanup created resources (keep namespace intact, minimal perms)
NS=${TEST_NS:-clowder-e2e}
echo "Cleaning up resources created by the test in namespace $NS"
# Delete ClowdEnvironment
oc delete clowdenvironment -n "$NS" $(oc get clowdenvironment -n "$NS" | grep test-basic-app | awk '{print $1}') --wait=true --ignore-not-found=true || true
# Delete ClowdApps (operator should GC owned resources)
oc -n "$NS" delete clowdapp --all --ignore-not-found=true || true
# Delete known test Secret from our resources (pull secret)
oc delete secret -n "$NS" $(oc get secret -n "$NS" | grep test-basic-app | awk '{print $1}') --ignore-not-found=true || true

exit $rc
