#!/bin/bash

set -euo pipefail

# Migration Helper Script: Static to Dynamic EC2 E2E Testing
# This script helps migrate from static EC2 instances to dynamic provisioning

echo "*** Clowder E2E Testing Migration Helper ***"
echo "This script will help you migrate from static to dynamic EC2 provisioning."
echo ""

# Check if we're in the right directory
if [ ! -f "ci/konflux_minikube_e2e_tests.sh" ]; then
    echo "Error: This script must be run from the clowder repository root"
    exit 1
fi

echo "Step 1: Checking current configuration..."

# Check if old environment variables are set
OLD_VARS_FOUND=false
if [ -n "${MINIKUBE_HOST:-}" ]; then
    echo "  Found old variable: MINIKUBE_HOST=$MINIKUBE_HOST"
    OLD_VARS_FOUND=true
fi
if [ -n "${MINIKUBE_USER:-}" ]; then
    echo "  Found old variable: MINIKUBE_USER=$MINIKUBE_USER"
    OLD_VARS_FOUND=true
fi
if [ -n "${MINIKUBE_SSH_KEY:-}" ]; then
    echo "  Found old variable: MINIKUBE_SSH_KEY (content hidden)"
    OLD_VARS_FOUND=true
fi
if [ -n "${MINIKUBE_ROOTDIR:-}" ]; then
    echo "  Found old variable: MINIKUBE_ROOTDIR=$MINIKUBE_ROOTDIR"
    OLD_VARS_FOUND=true
fi

if [ "$OLD_VARS_FOUND" = "true" ]; then
    echo "  ⚠️  Old static instance variables detected"
else
    echo "  ✓ No old static instance variables found"
fi

echo ""
echo "Step 2: Checking new AWS configuration..."

# Check if new environment variables are set
NEW_VARS_COMPLETE=true
REQUIRED_NEW_VARS=(
    "AWS_REGION"
    "EC2_KEY_PAIR_NAME"
    "EC2_SECURITY_GROUP_ID"
    "EC2_SUBNET_ID"
)

for var in "${REQUIRED_NEW_VARS[@]}"; do
    if [ -n "${!var:-}" ]; then
        echo "  ✓ $var is set"
    else
        echo "  ❌ $var is not set"
        NEW_VARS_COMPLETE=false
    fi
done

# Check private key configuration
if [ -n "${EC2_PRIVATE_KEY_PATH:-}" ]; then
    echo "  ✓ EC2_PRIVATE_KEY_PATH is set"
    if [ -f "${EC2_PRIVATE_KEY_PATH}" ]; then
        echo "    ✓ Private key file exists"
    else
        echo "    ❌ Private key file not found: $EC2_PRIVATE_KEY_PATH"
        NEW_VARS_COMPLETE=false
    fi
elif [ -n "${EC2_PRIVATE_KEY_CONTENT:-}" ]; then
    echo "  ✓ EC2_PRIVATE_KEY_CONTENT is set"
else
    echo "  ❌ Neither EC2_PRIVATE_KEY_PATH nor EC2_PRIVATE_KEY_CONTENT is set"
    NEW_VARS_COMPLETE=false
fi

echo ""
echo "Step 3: Checking AWS CLI and permissions..."

# Check AWS CLI
if command -v aws >/dev/null 2>&1; then
    echo "  ✓ AWS CLI is installed"
    
    # Test AWS credentials
    if aws sts get-caller-identity >/dev/null 2>&1; then
        echo "  ✓ AWS credentials are configured"
        
        # Test EC2 permissions
        if aws ec2 describe-instances --region "${AWS_REGION:-us-east-1}" --max-items 1 >/dev/null 2>&1; then
            echo "  ✓ EC2 permissions are working"
        else
            echo "  ❌ EC2 permissions test failed"
            NEW_VARS_COMPLETE=false
        fi
    else
        echo "  ❌ AWS credentials are not configured"
        NEW_VARS_COMPLETE=false
    fi
else
    echo "  ❌ AWS CLI is not installed"
    NEW_VARS_COMPLETE=false
fi

echo ""
echo "Step 4: Migration recommendations..."

if [ "$OLD_VARS_FOUND" = "true" ] && [ "$NEW_VARS_COMPLETE" = "true" ]; then
    echo "  🎉 Ready to migrate! Both old and new configurations are present."
    echo ""
    echo "  Next steps:"
    echo "  1. Test the new dynamic system:"
    echo "     ./ci/test_dynamic_provisioning.sh"
    echo ""
    echo "  2. Run a test E2E with dynamic provisioning:"
    echo "     ./ci/konflux_minikube_e2e_tests_dynamic.sh"
    echo ""
    echo "  3. Update your Tekton pipeline configuration:"
    echo "     - Replace ci/konflux_minikube_e2e_tests.sh with ci/konflux_minikube_e2e_tests_dynamic.sh"
    echo "     - Update secret from 'minikube-ssh-key' to 'aws-ec2-config'"
    echo "     - See ci/MIGRATION_GUIDE.md for detailed instructions"
    echo ""
    echo "  4. After successful testing, remove old environment variables"

elif [ "$NEW_VARS_COMPLETE" = "true" ]; then
    echo "  ✅ New dynamic configuration is complete!"
    echo ""
    echo "  You can start using dynamic provisioning:"
    echo "  ./ci/konflux_minikube_e2e_tests_dynamic.sh"

elif [ "$OLD_VARS_FOUND" = "true" ]; then
    echo "  ⚠️  Still using old static configuration."
    echo ""
    echo "  To migrate, you need to set up:"
    for var in "${REQUIRED_NEW_VARS[@]}"; do
        if [ -z "${!var:-}" ]; then
            echo "  - $var"
        fi
    done
    if [ -z "${EC2_PRIVATE_KEY_PATH:-}" ] && [ -z "${EC2_PRIVATE_KEY_CONTENT:-}" ]; then
        echo "  - EC2_PRIVATE_KEY_PATH or EC2_PRIVATE_KEY_CONTENT"
    fi
    echo ""
    echo "  See ci/README_dynamic_e2e.md for setup instructions."

else
    echo "  ❓ No E2E testing configuration found."
    echo ""
    echo "  To set up dynamic E2E testing, see ci/README_dynamic_e2e.md"
fi

echo ""
echo "Step 5: Available scripts..."
echo "  📖 ci/README_dynamic_e2e.md - Complete setup guide"
echo "  📖 ci/MIGRATION_GUIDE.md - Detailed migration instructions"
echo "  🧪 ci/test_dynamic_provisioning.sh - Validate configuration"
echo "  🚀 ci/provision_ec2_minikube.sh - Provision instance only"
echo "  🧹 ci/cleanup_ec2_minikube.sh - Cleanup instances"
echo "  🔄 ci/konflux_minikube_e2e_tests_dynamic.sh - Full dynamic E2E tests"

echo ""
echo "Migration helper completed!"
