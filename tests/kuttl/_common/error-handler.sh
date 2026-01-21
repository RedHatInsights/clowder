#!/bin/bash

# Common error handling for KUTTL tests
# This script should be sourced at the beginning of each json-assert test script
#
# Usage:
#   source "$(dirname "$0")/../_common/error-handler.sh"
#   setup_error_handling "test-name" "namespace"
#

# Function to collect events from a namespace
collect_namespace_events() {
    local ns="$1"
    echo "Collecting events for namespace: ${ns}" >&2

    # Check if namespace exists before trying to get events
    if kubectl get namespace "${ns}" >/dev/null 2>&1; then
        kubectl get events \
            --namespace="${ns}" \
            --sort-by='.metadata.creationTimestamp' \
            > "artifacts/kuttl/${KUTTL_TEST_NAME}/events-${ns}.txt" 2>&1 || true

        echo "Events saved to artifacts/kuttl/${KUTTL_TEST_NAME}/events-${ns}.txt" >&2
    else
        echo "Namespace ${ns} does not exist (yet), skipping event collection" >&2
    fi
}

# Function to collect events on failure
collect_events_on_failure() {
    exit_code=$?
    if [ $exit_code -ne 0 ]; then
        echo "Test failed with exit code $exit_code, collecting Kubernetes events..." >&2

        # Create artifacts directory if it doesn't exist
        mkdir -p "artifacts/kuttl/${KUTTL_TEST_NAME}"

        # Find all namespaces defined in the test's 00-install.yaml
        if [ -f "00-install.yaml" ]; then
            # Extract namespace names from 00-install.yaml
            NAMESPACES=$(grep -A2 "kind: Namespace" "00-install.yaml" | grep "name:" | awk '{print $2}' | sort -u)

            if [ -n "${NAMESPACES}" ]; then
                # Collect events from each namespace
                while IFS= read -r ns; do
                    collect_namespace_events "${ns}"
                done <<< "${NAMESPACES}"
            else
                echo "No namespaces found in 00-install.yaml, using fallback" >&2
                collect_namespace_events "${KUTTL_NAMESPACE}"
            fi
        else
            # Fallback: collect from the single namespace passed to setup
            echo "00-install.yaml not found, using fallback namespace" >&2
            collect_namespace_events "${KUTTL_NAMESPACE}"
        fi
    fi
    exit $exit_code
}

# Setup error handling for a test
# Arguments:
#   $1: Test name (used for artifacts directory)
#   $2: Namespace to collect events from (fallback if 00-install.yaml not found)
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
