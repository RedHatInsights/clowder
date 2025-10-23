# Dynamic EC2 Minikube E2E Testing

This directory contains scripts for running Clowder E2E tests on dynamically provisioned EC2 instances instead of using a long-standing instance.

## Overview

The new setup provides:
- **Fresh environment** for each test run
- **Better isolation** between test runs
- **Automatic cleanup** to prevent resource leaks
- **Cost optimization** by only running instances when needed

## Scripts

### `provision_ec2_minikube.sh`
Provisions a new EC2 instance with Minikube installed and configured.

### `cleanup_ec2_minikube.sh`
Terminates EC2 instances created for E2E testing.

### `konflux_minikube_e2e_tests_dynamic.sh`
Main E2E test script that integrates provisioning, testing, and cleanup.

## Required Environment Variables

### AWS Configuration
```bash
# AWS region for EC2 instances
AWS_REGION="us-east-1"

# EC2 instance configuration
EC2_KEY_PAIR_NAME="your-key-pair-name"
EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
EC2_SUBNET_ID="subnet-xxxxxxxxx"
EC2_PRIVATE_KEY_PATH="/path/to/your/private/key.pem"

# For Tekton/CI environments, use this instead of EC2_PRIVATE_KEY_PATH
EC2_PRIVATE_KEY_CONTENT="-----BEGIN RSA PRIVATE KEY-----\n..."
```

### Optional Configuration
```bash
# EC2 instance type (default: m5.2xlarge)
EC2_INSTANCE_TYPE="m5.2xlarge"

# AMI ID (default: Amazon Linux 2)
EC2_AMI_ID="ami-0c02fb55956c7d316"

# Minikube version (default: v1.34.0)
MINIKUBE_VERSION="v1.34.0"

# Kubernetes version (default: 1.30)
KUBERNETES_VERSION="1.30"

# Whether to wait for instance termination during cleanup (default: true)
WAIT_FOR_TERMINATION="true"
```

## AWS Infrastructure Requirements

### 1. VPC and Networking
- A VPC with internet gateway
- A public subnet with auto-assign public IP enabled
- Route table configured for internet access

### 2. Security Group
Create a security group with the following rules:
```
Inbound Rules:
- SSH (22): Your IP or CI/CD IP ranges
- Custom TCP (8443): Your IP or CI/CD IP ranges (for Kubernetes API)

Outbound Rules:
- All traffic (0.0.0.0/0) - for downloading packages and images
```

### 3. EC2 Key Pair
- Create an EC2 key pair in your target region
- Store the private key securely (for CI/CD, use secrets management)

### 4. IAM Permissions
The AWS credentials used must have permissions for:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "ec2:RunInstances",
                "ec2:TerminateInstances",
                "ec2:DescribeInstances",
                "ec2:DescribeInstanceStatus",
                "ec2:CreateTags"
            ],
            "Resource": "*"
        }
    ]
}
```

## Usage

### Local Testing
```bash
# Set up environment variables
export AWS_REGION="us-east-1"
export EC2_KEY_PAIR_NAME="my-key-pair"
export EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
export EC2_SUBNET_ID="subnet-xxxxxxxxx"
export EC2_PRIVATE_KEY_PATH="/path/to/key.pem"

# Run the dynamic E2E tests
./ci/konflux_minikube_e2e_tests_dynamic.sh
```

### CI/CD Integration (Tekton)
Update your Tekton pipeline to use the new script and provide the required environment variables through secrets.

Example secret configuration:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-ec2-config
type: Opaque
stringData:
  AWS_REGION: "us-east-1"
  EC2_KEY_PAIR_NAME: "clowder-ci-key"
  EC2_SECURITY_GROUP_ID: "sg-xxxxxxxxx"
  EC2_SUBNET_ID: "subnet-xxxxxxxxx"
  EC2_PRIVATE_KEY_CONTENT: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```

## Migration from Static Instance

### Tekton Pipeline Changes
1. Replace environment variables:
   - Remove: `MINIKUBE_HOST`, `MINIKUBE_USER`, `MINIKUBE_SSH_KEY`, `MINIKUBE_ROOTDIR`
   - Add: AWS configuration variables listed above

2. Update the script reference:
   ```bash
   # Old
   ci/konflux_minikube_e2e_tests.sh
   
   # New
   ci/konflux_minikube_e2e_tests_dynamic.sh
   ```

3. Update secret references in your pipeline configuration

### Benefits of Migration
- **Consistency**: Each test run starts with a clean environment
- **Isolation**: No interference between concurrent test runs
- **Reliability**: No dependency on long-running infrastructure
- **Cost**: Pay only for compute time during tests
- **Security**: Reduced attack surface with ephemeral instances

## Troubleshooting

### Instance Provisioning Issues
- Verify AWS credentials and permissions
- Check VPC/subnet configuration
- Ensure security group allows SSH access
- Verify AMI ID is valid for your region

### Cleanup Issues
- Check AWS permissions for EC2 termination
- Verify instance tags are set correctly
- Use manual cleanup: `./ci/cleanup_ec2_minikube.sh <instance-id> <region>`

### Test Failures
- Check instance logs: SSH to instance and check `/var/log/cloud-init-output.log`
- Verify Minikube status: `ssh -i key.pem ec2-user@<ip> minikube status`
- Check security group rules for port 8443 access

## Cost Optimization

The dynamic approach typically reduces costs by:
- Eliminating 24/7 instance costs
- Using instances only during test execution
- Automatic cleanup prevents forgotten resources
- Right-sizing instances for test requirements

Estimated cost savings: 80-90% compared to persistent instances (assuming tests run 2-3 hours per day).
