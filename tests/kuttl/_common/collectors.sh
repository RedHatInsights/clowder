#!/bin/bash

# Shared collection functions for KUTTL tests
# This library provides common functions for collecting Kubernetes resources
# and events when tests fail or when KUTTL collectors are invoked.
#
# Usage:
#   source "$(dirname "$0")/../_common/collectors.sh"
#

# Function to collect events from a namespace
collect_namespace_events() {
    set +x
    local ns="$1"
    local base_dir="${ARTIFACTS_DIR:-artifacts}"
    local artifacts_path="${base_dir}/kuttl/${KUTTL_TEST_NAME:-${TEST}}"

    echo "*** Collecting events for namespace: ${ns}" >&2

    # Check if namespace exists before trying to get events
    if kubectl get namespace "${ns}" >/dev/null 2>&1; then
        kubectl get events \
            --namespace="${ns}" \
            --sort-by='.metadata.creationTimestamp' \
            > "${artifacts_path}/events-${ns}.txt" 2>&1 || true

        echo "*** Events saved to ${artifacts_path}/events-${ns}.txt" >&2
    else
        echo "*** Namespace ${ns} does not exist (yet), skipping event collection" >&2
    fi
}

# Function to collect all resources from a namespace
collect_namespace_resources() {
    set +x
    local ns="$1"
    local base_dir="${ARTIFACTS_DIR:-artifacts}"
    local artifacts_path="${base_dir}/kuttl/${KUTTL_TEST_NAME:-${TEST}}"

    echo "*** Collecting all resources for namespace: ${ns}" >&2

    # Check if namespace exists before trying to get resources
    if kubectl get namespace "${ns}" >/dev/null 2>&1; then
        kubectl api-resources --verbs=list --namespaced -o name 2>/dev/null | \
            xargs -n 1 kubectl get --ignore-not-found -n "${ns}" -o yaml \
            > "${artifacts_path}/${ns}-resources.yaml" 2>&1 || true

        echo "*** Resources saved to ${artifacts_path}/${ns}-resources.yaml" >&2
    else
        echo "*** Namespace ${ns} does not exist (yet), skipping resource collection" >&2
    fi
}

# Function to collect ClowdEnvironment resources
collect_clowdenvironments() {
    set +x
    local test_dir="${1:-.}"
    local base_dir="${ARTIFACTS_DIR:-artifacts}"
    local artifacts_path="${base_dir}/kuttl/${KUTTL_TEST_NAME:-${TEST}}"

    echo "*** Collecting ClowdEnvironment resources..." >&2

    # Find all YAML files in the test directory
    local yaml_files=$(find "${test_dir}" -maxdepth 1 -name "*.yaml" -o -name "*.yml" 2>/dev/null)

    if [ -n "${yaml_files}" ]; then
        # Extract ClowdEnvironment names from all YAML files
        local clowdenvs=$(grep -h "kind: ClowdEnvironment" ${yaml_files} 2>/dev/null | \
            grep -A2 "kind: ClowdEnvironment" ${yaml_files} 2>/dev/null | \
            grep "name:" | awk '{print $2}' | sort -u)

        if [ -n "${clowdenvs}" ]; then
            # Collect each ClowdEnvironment
            while IFS= read -r env_name; do
                if [ -n "${env_name}" ] && kubectl get clowdenvironment "${env_name}" >/dev/null 2>&1; then
                    echo "*** Collecting ClowdEnvironment: ${env_name}" >&2
                    kubectl get clowdenvironment "${env_name}" -o yaml \
                        > "${artifacts_path}/clowdenvironment-${env_name}.yaml" 2>&1 || true
                    echo "*** ClowdEnvironment saved to ${artifacts_path}/clowdenvironment-${env_name}.yaml" >&2
                fi
            done <<< "${clowdenvs}"
        else
            echo "*** No ClowdEnvironment resources found in test files" >&2
        fi
    fi
}

# Function to find and collect from all test namespaces
collect_from_all_namespaces() {
    set +x
    local test_dir="${1:-.}"

    # Find all namespaces defined in the test's 00-install.yaml
    if [ -f "${test_dir}/00-install.yaml" ]; then
        # Extract namespace names from 00-install.yaml
        local namespaces=$(grep -A2 "kind: Namespace" "${test_dir}/00-install.yaml" | grep "name:" | awk '{print $2}' | sort -u)

        if [ -n "${namespaces}" ]; then
            # Collect events and resources from each namespace
            while IFS= read -r ns; do
                collect_namespace_events "${ns}"
                collect_namespace_resources "${ns}"
            done <<< "${namespaces}"
        else
            echo "*** No namespaces found in ${test_dir}/00-install.yaml" >&2
        fi
    else
        # Use NAMESPACE environment variable set by KUTTL
        echo "*** 00-install.yaml not found, using NAMESPACE environment variable" >&2
        if [ -n "${NAMESPACE}" ]; then
            collect_namespace_events "${NAMESPACE}"
            collect_namespace_resources "${NAMESPACE}"
        else
            echo "*** NAMESPACE environment variable not set, skipping namespace collection" >&2
        fi
    fi
}
