#!/usr/bin/env bash
set -euo pipefail

echo "Running tests"
# Ensure deploy and tests run
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

"$SCRIPT_DIR/deploy.sh"

# Run tests
set +e
pytest -q "$SCRIPT_DIR/tests"
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
