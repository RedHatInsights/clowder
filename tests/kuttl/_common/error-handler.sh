#!/bin/bash

# Common error handling for KUTTL tests
# This script should be sourced at the beginning of each json-assert test script
#
# Usage:
#   source "$(dirname "$0")/../_common/error-handler.sh"
#   setup_error_handling "test-name" "namespace"
#

# Function to collect events on failure
collect_events_on_failure() {
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "Test failed with exit code $exit_code, collecting Kubernetes events..." >&2

        # Create artifacts directory if it doesn't exist
        mkdir -p "kuttl_artifacts/${KUTTL_TEST_NAME}"

        # Collect events from the test namespace
        kubectl get events \
            --namespace="${KUTTL_NAMESPACE}" \
            --sort-by='.metadata.creationTimestamp' \
            > "kuttl_artifacts/${KUTTL_TEST_NAME}/events-${KUTTL_NAMESPACE}.txt" 2>&1 || true

        echo "Events saved to kuttl_artifacts/${KUTTL_TEST_NAME}/events-${KUTTL_NAMESPACE}.txt" >&2
    fi
    exit $exit_code
}

# Setup error handling for a test
# Arguments:
#   $1: Test name (used for artifacts directory)
#   $2: Namespace to collect events from
setup_error_handling() {
    if [ -z "$1" ] || [ -z "$2" ]; then
        echo "Error: setup_error_handling requires test name and namespace arguments" >&2
        exit 1
    fi

    export KUTTL_TEST_NAME="$1"
    export KUTTL_NAMESPACE="$2"

    # Enable strict error handling
    set -e
    set -o pipefail

    # Set trap to collect events on any error
    trap collect_events_on_failure EXIT
}
