#!/bin/bash

# Common error handling for KUTTL tests
# This script should be sourced at the beginning of each json-assert test script
#
# Usage:
#   source "$(dirname "$0")/../_common/error-handler.sh"
#   setup_error_handling "test-name"
#

# Source the shared collection functions
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/collectors.sh"

# Function to collect events on failure
collect_events_on_failure() {
    set +x
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "Test failed with exit code $exit_code, collecting Kubernetes events and resources..." >&2

        # Create artifacts directory if it doesn't exist
        local base_dir="${ARTIFACTS_DIR:-artifacts}"
        mkdir -p "${base_dir}/kuttl/${KUTTL_TEST_NAME}"

        # Collect from all namespaces
        collect_from_all_namespaces "."

        # Collect ClowdEnvironment resources
        collect_clowdenvironments "."
    fi
    exit $exit_code
}

# Setup error handling for a test
# Arguments:
#   $1: Test name (used for artifacts directory)
setup_error_handling() {
    if [ -z "$1" ]; then
        echo "Error: setup_error_handling requires test name argument" >&2
        exit 1
    fi

    export KUTTL_TEST_NAME="$1"

    # Enable strict error handling
    set -e
    set -o pipefail

    # Set trap to collect events on any error
    trap collect_events_on_failure EXIT
}
