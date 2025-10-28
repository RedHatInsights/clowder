#!/usr/bin/env bash
set -euo pipefail

# Ensure deploy and tests run
SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

"$SCRIPT_DIR/deploy.sh"

# Run tests
pytest -q "$SCRIPT_DIR/tests"
