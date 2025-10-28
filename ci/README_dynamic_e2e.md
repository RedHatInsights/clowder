# Dynamic EC2 Minikube E2E Testing

This directory contains scripts for running Clowder E2E tests on dynamically provisioned EC2 instances using a pre-built AMI with Minikube.

## Overview

The new setup provides:
- **Fresh environment** for each test run using your pre-built AMI
- **Faster startup** since Minikube is already installed in the AMI
- **Better isolation** between test runs
- **Automatic cleanup** to prevent resource leaks
- **Cost optimization** by only running instances when needed

## Scripts

### `provision_ec2_minikube.sh`
Provisions a new EC2 instance from your pre-built AMI with Minikube and starts a fresh Minikube cluster.

### `cleanup_ec2_minikube.sh`
Terminates EC2 instances created for E2E testing.

### `konflux_minikube_e2e_tests_dynamic.sh`
Main E2E test script that integrates provisioning, testing, and cleanup.

## Required Environment Variables

### AWS Configuration
```bash
# AWS region for EC2 instances
AWS_REGION="us-east-1"

# Your pre-built AMI ID with Minikube installed
EC2_AMI_ID="ami-xxxxxxxxx"

# EC2 instance configuration
EC2_KEY_PAIR_NAME="your-key-pair-name"
EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
EC2_SUBNET_ID="subnet-xxxxxxxxx"

# For Tekton/CI environments, use this instead of EC2_PRIVATE_KEY_PATH
EC2_PRIVATE_KEY_CONTENT="-----BEGIN RSA PRIVATE KEY-----\n..."
```

### Optional Configuration
```bash
# EC2 instance type (default: m5.2xlarge)
EC2_INSTANCE_TYPE="m5.2xlarge"

# Kubernetes version (default: 1.30)
KUBERNETES_VERSION="1.30"

# Whether to wait for instance termination during cleanup (default: true)
WAIT_FOR_TERMINATION="true"
```

## AWS Infrastructure Requirements

### 1. Pre-built AMI
- Create an AMI with Minikube, Docker, and kubectl pre-installed
- Ensure the AMI is based on Amazon Linux 2 or similar
- The `ec2-user` should have docker permissions

### 2. VPC and Networking
- A VPC with internet gateway
- A public subnet with auto-assign public IP enabled
- Route table configured for internet access

### 3. Security Group
Create a security group with the following rules:
```
Inbound Rules:
- SSH (22): Your IP or CI/CD IP ranges
- Custom TCP (8443): Your IP or CI/CD IP ranges (for Kubernetes API)

Outbound Rules:
- All traffic (0.0.0.0/0) - for downloading packages and images
```

### 4. EC2 Key Pair
- Create an EC2 key pair in your target region
- Store the private key securely (for CI/CD, use secrets management)

### 5. IAM Permissions
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
export EC2_AMI_ID="ami-xxxxxxxxx"
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
  EC2_AMI_ID: "ami-xxxxxxxxx"
  EC2_KEY_PAIR_NAME: "clowder-ci-key"
  EC2_SECURITY_GROUP_ID: "sg-xxxxxxxxx"
  EC2_SUBNET_ID: "subnet-xxxxxxxxx"
  EC2_PRIVATE_KEY_CONTENT: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```

## Benefits

- **Consistency**: Each test run starts with a clean environment
- **Speed**: Faster startup since Minikube is pre-installed
- **Cost Savings**: Only pay for compute time during test runs
- **Better Reliability**: Fresh environment eliminates accumulated state issues
- **Improved Security**: Instances are terminated after each run
- **Easier Debugging**: Each test run is isolated and reproducible

## Troubleshooting

### Common Issues

1. **AWS Permissions**: Ensure IAM user/role has required EC2 permissions
2. **Network Configuration**: Verify security group and subnet settings
3. **Key Pair**: Ensure EC2 key pair exists in the target region
4. **AMI Issues**: Verify your AMI has Minikube and Docker properly installed
5. **Instance Limits**: Check AWS service limits for EC2 instances

### Getting Help

- Check the logs in `/var/workdir/artifacts/` for detailed error information
- Verify your AMI by manually launching an instance and testing Minikube
- Review the cleanup logs to ensure instances are properly terminated

## Cost Considerations

The dynamic approach reduces costs because:
- Instances only run during test execution (~30-45 minutes)
- No idle time costs for long-running instances
- Automatic cleanup prevents forgotten instances
- Pre-built AMI reduces startup time

Estimated cost comparison (us-east-1, m5.2xlarge):
- **Static Instance**: $0.384/hour × 24 hours × 30 days = ~$276/month
- **Dynamic Instance**: $0.384/hour × 1 hour × 30 test runs = ~$12/month

*Actual costs may vary based on usage patterns and AWS pricing changes.*