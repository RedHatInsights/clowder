#!/usr/bin/env bash
set -euo pipefail

echo "Running tests"
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

# Cleanup namespace on success
if [ $rc -eq 0 ]; then
  if [ -n "${TEST_NS:-}" ]; then
    echo "Cleaning up namespace $TEST_NS"
    oc delete namespace "$TEST_NS" --wait=true --ignore-not-found=true
  fi
fi

exit $rc
