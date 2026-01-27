#!/bin/bash

# Run a script with output suppression
# On success: suppress all output
# On failure: display captured output and exit with error code
#
# Usage:
#   bash ../_common/run-script.sh <script-path>
#
# Example:
#   bash ../_common/run-script.sh 02-json-assert.sh

if [ -z "$1" ]; then
    echo "Error: run-script.sh requires a script path argument" >&2
    exit 1
fi

SCRIPT_PATH="$1"

if [ ! -f "$SCRIPT_PATH" ]; then
    echo "Error: Script not found: $SCRIPT_PATH" >&2
    exit 1
fi

# Capture all output (including set -x traces). On success, suppress output. On failure, display output and exit with error code.
OUTPUT=$(bash "$SCRIPT_PATH" 2>&1) || {
    echo "Assertion hit errors, see script output below"
    echo "$OUTPUT"
    exit 1
}
