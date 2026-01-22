#!/bin/bash

# Collector script for KUTTL tests to collect Kubernetes events on failure
# This script is called by KUTTL collectors when a test step fails
#
# Usage:
#   Called automatically by KUTTL with environment variables:
#   - NAMESPACE: The namespace being tested (set by KUTTL)
#   - TEST: The test name (set by KUTTL)
#

# Get test info from environment or current directory
TEST_NAME="${TEST:-$(basename "$(pwd)")}"
TEST_DIR="$(pwd)"

# Export as KUTTL_TEST_NAME for compatibility with collectors.sh
export KUTTL_TEST_NAME="${TEST_NAME}"

# Determine artifacts base directory and create it
BASE_DIR="${ARTIFACTS_DIR:-artifacts}"
ARTIFACTS_PATH="${BASE_DIR}/kuttl/${TEST_NAME}"
mkdir -p "${ARTIFACTS_PATH}"

# Source the shared collection functions
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "${SCRIPT_DIR}/collectors.sh"

# Collect from all namespaces
collect_from_all_namespaces "${TEST_DIR}"

# Collect ClowdEnvironment resources
collect_clowdenvironments "${TEST_DIR}"
