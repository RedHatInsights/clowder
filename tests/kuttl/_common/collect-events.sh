#!/bin/bash

# Collector script for KUTTL tests to collect Kubernetes events on failure
# This script is called by KUTTL collectors when a test step fails
#
# Usage:
#   Called automatically by KUTTL with environment variables:
#   - NAMESPACE: The namespace being tested (set by KUTTL)
#   - TEST: The test name (set by KUTTL)

# Get test info from environment or current directory
TEST_NAME="${TEST:-$(basename "$(pwd)")}"
TEST_DIR="$(pwd)"

# Determine artifacts base directory
BASE_DIR="${ARTIFACTS_DIR:-artifacts}"
ARTIFACTS_PATH="${BASE_DIR}/kuttl/${TEST_NAME}"
mkdir -p "${ARTIFACTS_PATH}"

# Function to collect events from a namespace
collect_namespace_events() {
    local ns="$1"
    echo "Collecting events for namespace: ${ns}" >&2

    # Check if namespace exists before trying to get events
    if kubectl get namespace "${ns}" >/dev/null 2>&1; then
        kubectl get events \
            --namespace="${ns}" \
            --sort-by='.metadata.creationTimestamp' \
            > "${ARTIFACTS_PATH}/events-${ns}.txt" 2>&1

        echo "Events saved to ${ARTIFACTS_PATH}/events-${ns}.txt" >&2
    else
        echo "Namespace ${ns} does not exist (yet), skipping event collection" >&2
    fi
}

# Find all namespaces defined in the test's 00-install.yaml
if [ -f "${TEST_DIR}/00-install.yaml" ]; then
    # Extract namespace names from 00-install.yaml
    NAMESPACES=$(grep -A2 "kind: Namespace" "${TEST_DIR}/00-install.yaml" | grep "name:" | awk '{print $2}' | sort -u)

    if [ -n "${NAMESPACES}" ]; then
        # Collect events from each namespace
        while IFS= read -r ns; do
            collect_namespace_events "${ns}"
        done <<< "${NAMESPACES}"
    else
        echo "No namespaces found in ${TEST_DIR}/00-install.yaml" >&2
    fi
else
    # Use NAMESPACE environment variable set by KUTTL
    echo "00-install.yaml not found, using NAMESPACE environment variable" >&2
    if [ -n "${NAMESPACE}" ]; then
        collect_namespace_events "${NAMESPACE}"
    else
        echo "NAMESPACE environment variable not set, skipping event collection" >&2
    fi
fi
