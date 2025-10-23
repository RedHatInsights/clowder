# Migration Guide: Static to Dynamic EC2 E2E Testing

This guide explains how to migrate from the static EC2 instance approach to the new dynamic EC2 provisioning system for Clowder E2E tests.

## Overview

The new dynamic system provides several advantages over the static instance approach:

- **Fresh Environment**: Each test run gets a clean EC2 instance
- **Better Isolation**: No interference between test runs
- **Cost Optimization**: Instances only run when needed
- **Automatic Cleanup**: No manual instance management required
- **Consistency**: Eliminates "works on my machine" issues

## Migration Steps

### 1. Update Environment Variables

Replace the old static instance variables with AWS configuration:

#### Remove (Old Static Instance Variables)
```bash
MINIKUBE_HOST="your-static-instance-ip"
MINIKUBE_USER="ec2-user"
MINIKUBE_SSH_KEY="-----BEGIN RSA PRIVATE KEY-----..."
MINIKUBE_ROOTDIR="/home/ec2-user"
```

#### Add (New AWS Configuration Variables)
```bash
# Required
AWS_REGION="us-east-1"
EC2_KEY_PAIR_NAME="your-key-pair-name"
EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
EC2_SUBNET_ID="subnet-xxxxxxxxx"
EC2_PRIVATE_KEY_CONTENT="-----BEGIN RSA PRIVATE KEY-----..."

# Optional (defaults provided)
EC2_INSTANCE_TYPE="m5.2xlarge"
EC2_AMI_ID="ami-0c02fb55956c7d316"  # Amazon Linux 2
MINIKUBE_VERSION="v1.34.0"
KUBERNETES_VERSION="1.30"
```

### 2. Update Tekton Pipeline Configuration

The main `clowder-pull-request.yaml` pipeline has been updated to use dynamic EC2 provisioning. The changes include:

#### E2E Test Task Changes

**Old Configuration (replaced):**
```yaml
- name: run-e2e-tests
  env:
  - name: MINIKUBE_HOST
    valueFrom:
      secretKeyRef:
        key: MINIKUBE_HOST
        name: minikube-ssh-key
  - name: MINIKUBE_USER
    valueFrom:
      secretKeyRef:
        key: MINIKUBE_USER
        name: minikube-ssh-key
  - name: MINIKUBE_SSH_KEY
    valueFrom:
      secretKeyRef:
        key: MINIKUBE_SSH_KEY
        name: minikube-ssh-key
  - name: MINIKUBE_ROOTDIR
    valueFrom:
      secretKeyRef:
        key: MINIKUBE_ROOTDIR
        name: minikube-ssh-key
  script: |
    ci/konflux_minikube_e2e_tests.sh
```

**New Configuration (now in main pipeline):**
```yaml
- name: run-e2e-tests
  env:
  - name: AWS_REGION
    valueFrom:
      secretKeyRef:
        key: AWS_REGION
        name: aws-ec2-config
  - name: EC2_KEY_PAIR_NAME
    valueFrom:
      secretKeyRef:
        key: EC2_KEY_PAIR_NAME
        name: aws-ec2-config
  - name: EC2_SECURITY_GROUP_ID
    valueFrom:
      secretKeyRef:
        key: EC2_SECURITY_GROUP_ID
        name: aws-ec2-config
  - name: EC2_SUBNET_ID
    valueFrom:
      secretKeyRef:
        key: EC2_SUBNET_ID
        name: aws-ec2-config
  - name: EC2_PRIVATE_KEY_CONTENT
    valueFrom:
      secretKeyRef:
        key: EC2_PRIVATE_KEY_CONTENT
        name: aws-ec2-config
  - name: AWS_ACCESS_KEY_ID
    valueFrom:
      secretKeyRef:
        key: AWS_ACCESS_KEY_ID
        name: aws-ec2-config
        optional: true
  - name: AWS_SECRET_ACCESS_KEY
    valueFrom:
      secretKeyRef:
        key: AWS_SECRET_ACCESS_KEY
        name: aws-ec2-config
        optional: true
  script: |
    # Install AWS CLI
    dnf install -y unzip curl
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
    unzip awscliv2.zip
    ./aws/install
    
    # Run dynamic E2E tests
    ci/konflux_minikube_e2e_tests_dynamic.sh
```

### 3. Create New Kubernetes Secret

**Old Secret (to be removed):**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: minikube-ssh-key
type: Opaque
stringData:
  MINIKUBE_HOST: "your-static-instance-ip"
  MINIKUBE_USER: "ec2-user"
  MINIKUBE_SSH_KEY: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
  MINIKUBE_ROOTDIR: "/home/ec2-user"
```

**New Secret:**
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
  # Optional: If not using IAM roles for AWS access
  AWS_ACCESS_KEY_ID: "AKIAIOSFODNN7EXAMPLE"
  AWS_SECRET_ACCESS_KEY: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
```

### 4. Pipeline Updates

The main `clowder-pull-request.yaml` pipeline has been updated to:
- Use the new `aws-ec2-config` secret instead of `minikube-ssh-key`
- Call `ci/konflux_minikube_e2e_tests_dynamic.sh` instead of the old static script
- Install AWS CLI v2 for EC2 provisioning

No separate pipeline file is needed - the changes are integrated into the existing pipeline.

### 5. AWS Infrastructure Setup

Ensure you have the required AWS infrastructure:

#### VPC and Networking
- VPC with internet gateway
- Public subnet with auto-assign public IP enabled
- Route table configured for internet access

#### Security Group Rules
```
Inbound Rules:
- SSH (22): CI/CD IP ranges
- Custom TCP (8443): CI/CD IP ranges (for Kubernetes API)

Outbound Rules:
- All traffic (0.0.0.0/0) - for downloading packages and images
```

#### IAM Permissions
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

## Testing the Migration

### 1. Validate Configuration
```bash
# Set environment variables
export AWS_REGION="us-east-1"
export EC2_KEY_PAIR_NAME="your-key-pair"
export EC2_SECURITY_GROUP_ID="sg-xxxxxxxxx"
export EC2_SUBNET_ID="subnet-xxxxxxxxx"
export EC2_PRIVATE_KEY_PATH="/path/to/key.pem"

# Run validation script
./ci/test_dynamic_provisioning.sh
```

### 2. Test Provisioning Only
```bash
# Test just the provisioning
./ci/provision_ec2_minikube.sh

# Cleanup when done
./ci/cleanup_ec2_minikube.sh
```

### 3. Full E2E Test
```bash
# Run complete dynamic E2E test
./ci/konflux_minikube_e2e_tests_dynamic.sh
```

## Rollback Plan

If you need to rollback to the static instance approach:

1. Revert the Tekton pipeline configuration
2. Restore the old `minikube-ssh-key` secret
3. Use the original script: `ci/konflux_minikube_e2e_tests.sh`
4. Ensure your static EC2 instance is running and accessible

## Benefits After Migration

- **Reduced Maintenance**: No need to manage long-running EC2 instances
- **Cost Savings**: Only pay for compute time during test runs
- **Better Reliability**: Fresh environment eliminates accumulated state issues
- **Improved Security**: Instances are terminated after each run
- **Easier Debugging**: Each test run is isolated and reproducible

## Troubleshooting

### Common Issues

1. **AWS Permissions**: Ensure IAM user/role has required EC2 permissions
2. **Network Configuration**: Verify security group and subnet settings
3. **Key Pair**: Ensure EC2 key pair exists in the target region
4. **Instance Limits**: Check AWS service limits for EC2 instances

### Getting Help

- Check the logs in `/var/workdir/artifacts/` for detailed error information
- Use the test script `ci/test_dynamic_provisioning.sh` to validate setup
- Review the cleanup logs to ensure instances are properly terminated

## Cost Considerations

The dynamic approach typically reduces costs because:
- Instances only run during test execution (~30-45 minutes)
- No idle time costs for long-running instances
- Automatic cleanup prevents forgotten instances

Estimated cost comparison (us-east-1, m5.2xlarge):
- **Static Instance**: $0.384/hour × 24 hours × 30 days = ~$276/month
- **Dynamic Instance**: $0.384/hour × 1 hour × 30 test runs = ~$12/month

*Actual costs may vary based on usage patterns and AWS pricing changes.*
