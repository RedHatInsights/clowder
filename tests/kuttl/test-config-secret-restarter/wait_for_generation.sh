#!/bin/bash

# Script to wait for a deployment's observedGeneration to reach or exceed the specified value
# Usage: ./wait_for_generation <deployment_name> <desired_generation>
# Example: ./wait_for_generation my-app 6

if [ $# -ne 2 ]; then
    echo "Usage: $0 <deployment_name> <desired_generation>"
    echo "Example: $0 my-app 6"
    exit 1
fi

DEPLOYMENT_NAME="$1"
DESIRED_GENERATION="$2"
NAMESPACE="test-config-secret-restarter"
MAX_ATTEMPTS=30
DELAY=2

echo "Waiting for deployment '$DEPLOYMENT_NAME' observedGeneration to be >= $DESIRED_GENERATION"

for attempt in $(seq 1 $MAX_ATTEMPTS); do
    # Get current observedGeneration
    current_generation=$(kubectl get deployment "$DEPLOYMENT_NAME" -n "$NAMESPACE" -o jsonpath='{.status.observedGeneration}' 2>/dev/null)

    if [ -z "$current_generation" ]; then
        echo "Attempt $attempt: Could not get observedGeneration for deployment '$DEPLOYMENT_NAME'"
    else
        # Check if current generation is greater than or equal to desired generation
        if [ "$current_generation" -ge "$DESIRED_GENERATION" ]; then
            echo "Success: Deployment '$DEPLOYMENT_NAME' observedGeneration is now: $current_generation (>= $DESIRED_GENERATION)"
            exit 0
        fi
        echo "Attempt $attempt: Deployment '$DEPLOYMENT_NAME' observedGeneration is $current_generation (waiting for >= $DESIRED_GENERATION)"
    fi

    if [ $attempt -lt $MAX_ATTEMPTS ]; then
        sleep $DELAY
    fi
done

echo "Timeout: Deployment '$DEPLOYMENT_NAME' observedGeneration did not reach the minimum expected value: $DESIRED_GENERATION"
echo "Final observedGeneration: $current_generation"
exit 1
