#!/bin/bash

set -euo pipefail

# Test script for dynamic EC2 provisioning
# This script validates that the provisioning and cleanup works correctly

echo "*** Testing Dynamic EC2 Provisioning for Clowder E2E ***"

# Check required environment variables
REQUIRED_VARS=(
    "AWS_REGION"
    "EC2_KEY_PAIR_NAME"
    "EC2_SECURITY_GROUP_ID"
    "EC2_SUBNET_ID"
    "EC2_PRIVATE_KEY_PATH"
)

echo "*** Checking required environment variables..."
for var in "${REQUIRED_VARS[@]}"; do
    if [ -z "${!var:-}" ]; then
        echo "ERROR: Required environment variable $var is not set"
        exit 1
    fi
    echo "✓ $var is set"
done

# Check AWS CLI is available and configured
echo "*** Checking AWS CLI..."
if ! command -v aws >/dev/null 2>&1; then
    echo "ERROR: AWS CLI is not installed"
    exit 1
fi

# Test AWS credentials
if ! aws sts get-caller-identity >/dev/null 2>&1; then
    echo "ERROR: AWS credentials are not configured or invalid"
    exit 1
fi
echo "✓ AWS CLI is configured and working"

# Check private key file
if [ ! -f "$EC2_PRIVATE_KEY_PATH" ]; then
    echo "ERROR: Private key file not found at $EC2_PRIVATE_KEY_PATH"
    exit 1
fi

if [ "$(stat -c %a "$EC2_PRIVATE_KEY_PATH")" != "600" ]; then
    echo "WARNING: Private key file permissions are not 600, fixing..."
    chmod 600 "$EC2_PRIVATE_KEY_PATH"
fi
echo "✓ Private key file is accessible"

# Test EC2 permissions
echo "*** Testing EC2 permissions..."
if ! aws ec2 describe-instances --region "$AWS_REGION" --max-items 1 >/dev/null 2>&1; then
    echo "ERROR: Cannot describe EC2 instances - check IAM permissions"
    exit 1
fi
echo "✓ EC2 describe permissions are working"

# Test security group exists
echo "*** Validating security group..."
if ! aws ec2 describe-security-groups --region "$AWS_REGION" --group-ids "$EC2_SECURITY_GROUP_ID" >/dev/null 2>&1; then
    echo "ERROR: Security group $EC2_SECURITY_GROUP_ID not found in region $AWS_REGION"
    exit 1
fi
echo "✓ Security group exists"

# Test subnet exists
echo "*** Validating subnet..."
if ! aws ec2 describe-subnets --region "$AWS_REGION" --subnet-ids "$EC2_SUBNET_ID" >/dev/null 2>&1; then
    echo "ERROR: Subnet $EC2_SUBNET_ID not found in region $AWS_REGION"
    exit 1
fi
echo "✓ Subnet exists"

# Test key pair exists
echo "*** Validating key pair..."
if ! aws ec2 describe-key-pairs --region "$AWS_REGION" --key-names "$EC2_KEY_PAIR_NAME" >/dev/null 2>&1; then
    echo "ERROR: Key pair $EC2_KEY_PAIR_NAME not found in region $AWS_REGION"
    exit 1
fi
echo "✓ Key pair exists"

echo ""
echo "*** All validation checks passed! ***"
echo ""
echo "You can now run the dynamic E2E tests with:"
echo "  ./ci/konflux_minikube_e2e_tests_dynamic.sh"
echo ""
echo "Or test just the provisioning with:"
echo "  ./ci/provision_ec2_minikube.sh"
echo "  ./ci/cleanup_ec2_minikube.sh"
echo ""

# Optional: Run a quick provisioning test
if [ "${RUN_PROVISIONING_TEST:-false}" = "true" ]; then
    echo "*** Running provisioning test (RUN_PROVISIONING_TEST=true)..."
    
    # Set a short instance name for testing
    export EC2_INSTANCE_NAME="clowder-e2e-test-$(date +%H%M%S)"
    
    echo "*** Provisioning test instance: $EC2_INSTANCE_NAME"
    source ./ci/provision_ec2_minikube.sh
    
    echo "*** Provisioning successful! Instance ID: $MINIKUBE_INSTANCE_ID"
    echo "*** Waiting 30 seconds before cleanup..."
    sleep 30
    
    echo "*** Cleaning up test instance..."
    ./ci/cleanup_ec2_minikube.sh
    
    echo "*** Provisioning test completed successfully!"
fi
