# Dynamic EC2 E2E Testing with Pre-built AMI

This directory contains scripts for running Clowder E2E tests on dynamically provisioned EC2 instances using a pre-built AMI with Minikube already installed.

## Overview

The new setup provides:
- **Fresh environment** for each test run using pre-built AMI
- **Faster provisioning** since Minikube is pre-installed
- **Better isolation** between test runs
- **Automatic cleanup** to prevent resource leaks
- **Cost optimization** by only running instances when needed

## Scripts

### `provision_ec2_minikube.sh`
Provisions a new EC2 instance from a pre-built AMI with Minikube already installed and configured.

### `cleanup_ec2_minikube.sh`
Terminates EC2 instances created for E2E testing with multiple cleanup methods.

### `konflux_minikube_e2e_tests_dynamic.sh`
Main E2E test script that integrates provisioning, testing, and cleanup using the pre-built AMI.

## Required Environment Variables

### AWS Configuration
```bash
# AWS region for EC2 instances
AWS_REGION="us-east-1"

# Pre-built AMI with Minikube installed
EC2_AMI_ID="ami-xxxxxxxxx"

# EC2 instance configuration
EC2_KEY_PAIR_NAME="your-key-pair-name"
EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
EC2_SUBNET_ID="subnet-xxxxxxxxx"

# For Tekton/CI environments, use this instead of EC2_PRIVATE_KEY_PATH
EC2_PRIVATE_KEY_CONTENT="-----BEGIN RSA PRIVATE KEY-----\n..."

# For local testing
EC2_PRIVATE_KEY_PATH="/path/to/your/private/key.pem"
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

## Pre-built AMI Requirements

The AMI must have the following pre-installed:
- **Docker** (latest stable version)
- **Minikube** (v1.34.0 or compatible)
- **kubectl** (compatible with Kubernetes 1.30)
- **conntrack** (required for minikube)
- **Basic utilities**: curl, wget, unzip, git

### Recommended AMI Setup
```bash
# Base: Amazon Linux 2 or Ubuntu 20.04+
# Install Docker
sudo yum install -y docker  # Amazon Linux 2
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker ec2-user

# Install kubectl
curl -LO "https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/

# Install Minikube
curl -LO https://storage.googleapis.com/minikube/releases/v1.34.0/minikube-linux-amd64
chmod +x minikube-linux-amd64
sudo mv minikube-linux-amd64 /usr/local/bin/minikube

# Install conntrack
sudo yum install -y conntrack  # Amazon Linux 2

# Prepare minikube directory
mkdir -p /home/ec2-user/.minikube
chown -R ec2-user:ec2-user /home/ec2-user/.minikube
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
                "ec2:CreateTags",
                "ec2:DescribeSecurityGroups",
                "ec2:DescribeSubnets",
                "ec2:DescribeKeyPairs"
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
- **Speed**: Faster provisioning with pre-built AMI
- **Reliability**: No accumulated state issues
- **Cost Efficiency**: Only pay for compute time during test runs
- **Scalability**: Can run multiple tests in parallel

## Troubleshooting

### Common Issues

1. **AMI Validation**: Ensure Minikube is properly installed on the AMI
2. **AWS Permissions**: Ensure IAM user/role has required EC2 permissions
3. **Network Configuration**: Verify security group and subnet settings
4. **Key Pair**: Ensure EC2 key pair exists in the target region
5. **Instance Limits**: Check AWS service limits for EC2 instances

### Getting Help

- Check the logs in `/var/workdir/artifacts/` for detailed error information
- Verify AMI has all required components installed
- Review the cleanup logs to ensure instances are properly terminated

## Cost Considerations

The dynamic approach with pre-built AMI provides optimal cost efficiency:
- Instances only run during test execution (~20-30 minutes with pre-built AMI)
- No idle time costs for long-running instances
- Automatic cleanup prevents forgotten instances
- Faster provisioning reduces overall test time

Estimated cost comparison (us-east-1, m5.2xlarge):
- **Static Instance**: $0.384/hour × 24 hours × 30 days = ~$276/month
- **Dynamic Instance (pre-built AMI)**: $0.384/hour × 0.5 hour × 30 test runs = ~$6/month

*Actual costs may vary based on usage patterns and AWS pricing changes.*
