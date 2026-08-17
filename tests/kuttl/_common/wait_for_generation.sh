#!/bin/bash

# Usage: ./wait_for_generation.sh <deployment_name> <desired_generation> [namespace]
# Example: ./wait_for_generation.sh my-app 6 test-my-feature

if [ $# -lt 2 ]; then
    echo "Usage: $0 <deployment_name> <desired_generation> [namespace]"
    echo "Example: $0 my-app 6 test-my-feature"
    exit 1
fi

DEPLOYMENT_NAME="$1"
DESIRED_GENERATION="$2"
NAMESPACE="${3:-default}"
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
