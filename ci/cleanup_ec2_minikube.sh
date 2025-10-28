#!/bin/bash

set -euo pipefail

# EC2 Minikube Instance Cleanup Script
# This script terminates the EC2 instance created for E2E testing

# Function to cleanup instance by ID
cleanup_instance() {
    local instance_id="$1"
    local region="$2"
    
    echo "*** Terminating EC2 instance: $instance_id in region: $region"
    
    # Check if instance exists and is not already terminated
    INSTANCE_STATE=$(aws ec2 describe-instances \
        --region "$region" \
        --instance-ids "$instance_id" \
        --query 'Reservations[0].Instances[0].State.Name' \
        --output text 2>/dev/null || echo "not-found")
    
    if [ "$INSTANCE_STATE" = "not-found" ]; then
        echo "*** Instance $instance_id not found, may have been already terminated"
        return 0
    elif [ "$INSTANCE_STATE" = "terminated" ] || [ "$INSTANCE_STATE" = "terminating" ]; then
        echo "*** Instance $instance_id is already $INSTANCE_STATE"
        return 0
    fi
    
    echo "*** Instance $instance_id is in state: $INSTANCE_STATE"
    
    # Terminate the instance
    aws ec2 terminate-instances \
        --region "$region" \
        --instance-ids "$instance_id" \
        --output text
    
    echo "*** Termination request sent for instance: $instance_id"
    
    # Optionally wait for termination (can be disabled for faster cleanup)
    if [ "${WAIT_FOR_TERMINATION:-true}" = "true" ]; then
        echo "*** Waiting for instance to terminate..."
        aws ec2 wait instance-terminated --region "$region" --instance-ids "$instance_id" || {
            echo "*** Warning: Timeout waiting for instance termination, but termination was requested"
        }
        echo "*** Instance $instance_id has been terminated"
    fi
}

# Method 1: Use environment variables if available
if [ -n "${MINIKUBE_INSTANCE_ID:-}" ] && [ -n "${AWS_REGION:-}" ]; then
    echo "*** Using environment variables for cleanup"
    cleanup_instance "$MINIKUBE_INSTANCE_ID" "$AWS_REGION"
    exit 0
fi

# Method 2: Use instance info file if available
if [ -f "/tmp/minikube-instance-info" ]; then
    echo "*** Loading instance info from /tmp/minikube-instance-info"
    source /tmp/minikube-instance-info
    
    if [ -n "${MINIKUBE_INSTANCE_ID:-}" ] && [ -n "${AWS_REGION:-}" ]; then
        cleanup_instance "$MINIKUBE_INSTANCE_ID" "$AWS_REGION"
        rm -f /tmp/minikube-instance-info
        exit 0
    else
        echo "*** Error: Instance info file exists but missing required variables"
        exit 1
    fi
fi

# Method 3: Use command line arguments
if [ $# -eq 2 ]; then
    INSTANCE_ID="$1"
    REGION="$2"
    echo "*** Using command line arguments for cleanup"
    cleanup_instance "$INSTANCE_ID" "$REGION"
    exit 0
fi

# Method 4: Find instances by tags (fallback)
echo "*** No instance ID provided, searching for clowder E2E instances..."
: ${AWS_REGION:="us-east-1"}

# Find instances tagged for clowder E2E testing that are not terminated
INSTANCE_IDS=$(aws ec2 describe-instances \
    --region "$AWS_REGION" \
    --filters \
        "Name=tag:Purpose,Values=clowder-e2e-testing" \
        "Name=tag:AutoDelete,Values=true" \
        "Name=instance-state-name,Values=pending,running,shutting-down,stopping,stopped" \
    --query 'Reservations[].Instances[].InstanceId' \
    --output text)

if [ -z "$INSTANCE_IDS" ]; then
    echo "*** No clowder E2E instances found to cleanup"
    exit 0
fi

echo "*** Found instances to cleanup: $INSTANCE_IDS"

# Cleanup each found instance
for instance_id in $INSTANCE_IDS; do
    cleanup_instance "$instance_id" "$AWS_REGION"
done

echo "*** Cleanup completed!"