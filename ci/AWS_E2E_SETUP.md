# AWS E2E Testing Setup Guide

This guide explains how to use the updated `konflux_minikube_e2e_tests.sh` script that can provision AWS EC2 instances for each pipeline run or fall back to the legacy minikube setup.

## Overview

The updated script now supports two modes:

### AWS Mode (New)
- Creates a dedicated AWS EC2 instance for each pipeline run
- Installs and configures minikube on the EC2 instance
- Runs E2E tests in complete isolation
- Automatically cleans up all AWS resources after completion

### Legacy Mode (Backward Compatible)
- Uses existing minikube infrastructure (original behavior)
- Maintains compatibility with current setups
- No AWS resources required

## Environment Variables

The script automatically detects which mode to use based on available environment variables:

### For AWS Mode
```bash
# Required
AWS_ACCESS_KEY_ID=your_access_key_id
AWS_SECRET_ACCESS_KEY=your_secret_access_key
AWS_REGION=us-east-1  # or your preferred region
```

### For Legacy Mode
```bash
# Required (existing setup)
MINIKUBE_HOST=your_minikube_host
MINIKUBE_USER=your_minikube_user
MINIKUBE_SSH_KEY=your_ssh_private_key
MINIKUBE_ROOTDIR=/path/to/minikube/root
```

### Optional AWS Configuration
```bash
# EC2 Instance Configuration
AWS_INSTANCE_TYPE=t3.xlarge          # Default: t3.xlarge
AWS_AMI_ID=ami-0c02fb55956c7d316     # Default: Amazon Linux 2023 (us-east-1)
AWS_SUBNET_ID=subnet-xxxxxxxxx       # Optional: Uses default VPC if not specified

# Pipeline Identification
TEKTON_PIPELINE_RUN=unique-run-id    # Optional: Auto-generated if not provided
```

## AWS Permissions Required

The AWS credentials must have the following permissions:

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
                "ec2:DescribeVpcs",
                "ec2:DescribeSubnets",
                "ec2:CreateKeyPair",
                "ec2:DeleteKeyPair",
                "ec2:CreateSecurityGroup",
                "ec2:DeleteSecurityGroup",
                "ec2:AuthorizeSecurityGroupIngress",
                "ec2:DescribeSecurityGroups"
            ],
            "Resource": "*"
        }
    ]
}
```

## How It Works

1. **AWS Resource Provisioning**:
   - Creates a unique key pair for SSH access
   - Creates a security group with SSH (22) and Kubernetes API (8443) access
   - Launches an EC2 instance with Docker and minikube pre-installed

2. **Minikube Setup**:
   - Connects to the EC2 instance via SSH
   - Starts minikube with optimized settings for AWS EC2
   - Configures port forwarding for Kubernetes API access

3. **Test Execution**:
   - Runs the existing E2E test suite against the minikube cluster
   - Collects logs and artifacts as before

4. **Cleanup**:
   - Automatically terminates the EC2 instance
   - Deletes the key pair and security group
   - Cleanup happens even if tests fail (via trap)

## Instance Specifications

### Default Instance Type: t3.xlarge
- 4 vCPUs
- 16 GB RAM
- Up to 5 Gbps network performance
- EBS-optimized

### Minikube Configuration
- Driver: Docker
- CPUs: 6 (uses available cores)
- Memory: 14GB (leaves 2GB for system)
- Disk: 20GB
- Kubernetes version: 1.30
- Addons: metrics-server

## Cost Considerations

- **t3.xlarge**: ~$0.1664/hour (us-east-1, on-demand)
- **EBS storage**: ~$0.10/GB/month for 20GB
- **Data transfer**: Minimal for E2E tests

Typical E2E test run: ~30-60 minutes = $0.08-$0.17 per run

## Troubleshooting

### Common Issues

1. **Instance launch fails**:
   - Check AWS credentials and permissions
   - Verify the AMI ID is valid for your region
   - Ensure you have EC2 service limits available

2. **SSH connection timeout**:
   - Security group rules may be incorrect
   - Instance may still be initializing
   - Check if the instance has a public IP

3. **Minikube start fails**:
   - Instance may not have enough resources
   - Docker service may not be running
   - Check instance logs via AWS Console

### Debugging

To debug issues, you can:

1. Check the pipeline logs for detailed error messages
2. Use AWS Console to inspect the created resources
3. SSH manually to the instance (if it's still running) using the generated key

### Manual Cleanup

If the script fails and doesn't clean up automatically:

```bash
# List instances with the tag pattern
aws ec2 describe-instances --filters "Name=key-name,Values=clowder-e2e-*"

# Terminate specific instance
aws ec2 terminate-instances --instance-ids i-xxxxxxxxx

# Delete key pair
aws ec2 delete-key-pair --key-name clowder-e2e-XXXXXXXXX

# Delete security group
aws ec2 delete-security-group --group-id sg-xxxxxxxxx
```

## Regional AMI IDs

The default AMI is for us-east-1. For other regions, update `AWS_AMI_ID`:

- **us-west-2**: ami-0c2d3e23b7e6c7c7c
- **eu-west-1**: ami-0c94855ba95b798c7
- **ap-southeast-1**: ami-0c802847a7dd848c0

(Use the latest Amazon Linux 2023 AMI for your region)

## Integration with Tekton Pipeline

To integrate with your Tekton pipeline, add the AWS environment variables to the pipeline task:

```yaml
env:
- name: AWS_ACCESS_KEY_ID
  valueFrom:
    secretKeyRef:
      key: aws_access_key_id
      name: aws-credentials
- name: AWS_SECRET_ACCESS_KEY
  valueFrom:
    secretKeyRef:
      key: aws_secret_access_key
      name: aws-credentials
- name: AWS_REGION
  value: "us-east-1"
```
