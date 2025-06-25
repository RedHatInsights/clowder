#!/bin/bash

# Script to wait for a deployment's observedGeneration to reach one of the specified values
# Usage: ./wait_for_generation <deployment_name> "<generation1> [generation2] [generation3] ..."
# Example: ./wait_for_generation my-app "6 7 8"

if [ $# -ne 2 ]; then
    echo "Usage: $0 <deployment_name> \"<generation1> [generation2] [generation3] ...\""
    echo "Example: $0 my-app \"6 7 8\""
    exit 1
fi

DEPLOYMENT_NAME="$1"
VALID_GENERATIONS="$2"
NAMESPACE="test-config-secret-restarter"
MAX_ATTEMPTS=30
DELAY=2

echo "Waiting for deployment '$DEPLOYMENT_NAME' observedGeneration to be one of: $VALID_GENERATIONS"

for attempt in $(seq 1 $MAX_ATTEMPTS); do
    # Get current observedGeneration
    current_generation=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.status.observedGeneration}' 2>/dev/null)

    if [ -z "$current_generation" ]; then
        echo "Attempt $attempt: Could not get observedGeneration for deployment '$DEPLOYMENT_NAME'"
    else
        # Check if current generation matches any of the valid generations
        for valid_gen in $VALID_GENERATIONS; do
            if [ "$current_generation" = "$valid_gen" ]; then
                echo "Success: Deployment '$DEPLOYMENT_NAME' observedGeneration is now: $current_generation"
                exit 0
            fi
        done
        echo "Attempt $attempt: Deployment '$DEPLOYMENT_NAME' observedGeneration is $current_generation (waiting for: $VALID_GENERATIONS)"
    fi

    if [ $attempt -lt $MAX_ATTEMPTS ]; then
        sleep $DELAY
    fi
done

echo "Timeout: Deployment '$DEPLOYMENT_NAME' observedGeneration did not reach any of the expected values: $VALID_GENERATIONS"
echo "Final observedGeneration: $current_generation"
exit 1
