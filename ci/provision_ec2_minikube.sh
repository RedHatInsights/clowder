#!/bin/bash

set -euo pipefail

# EC2 Minikube Instance Provisioning Script (Pre-built AMI)
# This script provisions a new EC2 instance from a pre-built AMI with Minikube already installed

# Required environment variables
: ${AWS_REGION:="us-east-1"}
: ${EC2_INSTANCE_TYPE:="m5.2xlarge"}  # 8 vCPUs, 32 GB RAM
: ${MINIKUBE_EC2_AMI_ID:?"MINIKUBE_EC2_AMI_ID must be set (pre-built AMI with Minikube)"}
# : ${EC2_KEY_PAIR_NAME:?"EC2_KEY_PAIR_NAME must be set"}
# : ${EC2_SECURITY_GROUP_ID:?"EC2_SECURITY_GROUP_ID must be set"}
# : ${EC2_SUBNET_ID:?"EC2_SUBNET_ID must be set"}

# Optional environment variables
: ${EC2_INSTANCE_NAME:="clowder-e2e-$(date +%Y%m%d-%H%M%S)-$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"}
: ${KUBERNETES_VERSION:="1.30"}

echo "*** Provisioning EC2 instance for Clowder E2E tests ***"
echo "Instance Name: $EC2_INSTANCE_NAME"
echo "Instance Type: $EC2_INSTANCE_TYPE"
echo "AMI ID: $MINIKUBE_EC2_AMI_ID"
echo "Region: $AWS_REGION"

# Launch EC2 instance from pre-built AMI
echo "*** Launching EC2 instance from pre-built AMI..."
INSTANCE_ID=$(aws ec2 run-instances \
    --region "$AWS_REGION" \
    --image-id "$MINIKUBE_EC2_AMI_ID" \
    --instance-type "$EC2_INSTANCE_TYPE" \
    # --key-name "$EC2_KEY_PAIR_NAME" \
    # --security-group-ids "$EC2_SECURITY_GROUP_ID" \
    # --subnet-id "$EC2_SUBNET_ID" \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$EC2_INSTANCE_NAME},{Key=Purpose,Value=clowder-e2e-testing},{Key=AutoDelete,Value=true},{Key=CreatedBy,Value=clowder-ci}]" \
    --query 'Instances[0].InstanceId' \
    --output text)

if [ -z "$INSTANCE_ID" ] || [ "$INSTANCE_ID" = "None" ]; then
    echo "*** Error: Failed to launch EC2 instance"
    exit 1
fi

echo "*** Instance launched with ID: $INSTANCE_ID"

# Wait for instance to be running
echo "*** Waiting for instance to be running..."
aws ec2 wait instance-running --region "$AWS_REGION" --instance-ids "$INSTANCE_ID"

# Get instance public IP
PUBLIC_IP=$(aws ec2 describe-instances \
    --region "$AWS_REGION" \
    --instance-ids "$INSTANCE_ID" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

if [ -z "$PUBLIC_IP" ] || [ "$PUBLIC_IP" = "None" ]; then
    echo "*** Error: Failed to get public IP for instance $INSTANCE_ID"
    echo "*** Cleaning up failed instance..."
    aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "$INSTANCE_ID" || true
    exit 1
fi

echo "*** Instance is running at IP: $PUBLIC_IP"

# Wait for SSH to be available
echo "*** Waiting for SSH to be available..."
SSH_READY=false
for i in {1..30}; do
    if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" echo "SSH is ready" 2>/dev/null; then
        echo "*** SSH is available"
        SSH_READY=true
        break
    fi
    echo "Attempt $i/30: SSH not ready, waiting 10 seconds..."
    sleep 10
done

if [ "$SSH_READY" != "true" ]; then
    echo "*** Error: SSH connection failed after 30 attempts"
    echo "*** Cleaning up failed instance: $INSTANCE_ID"
    aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "$INSTANCE_ID" || true
    exit 1
fi

# Verify Minikube is available on the AMI
echo "*** Verifying Minikube is available on the AMI..."
if ! ssh -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" "which minikube" 2>/dev/null; then
    echo "*** Error: Minikube not found on AMI. Please ensure the AMI has Minikube pre-installed."
    echo "*** Cleaning up failed instance: $INSTANCE_ID"
    aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "$INSTANCE_ID" || true
    exit 1
fi

# Start Minikube on the pre-built AMI
echo "*** Starting Minikube on EC2 instance..."
if ! ssh -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" << EOF
    set -euo pipefail
    
    # Clean up any existing minikube state
    minikube delete || true
    
    # Start Minikube with the same configuration as the original script
    minikube start \
        --cpus 6 \
        --disk-size 10GB \
        --memory 16000MB \
        --kubernetes-version=$KUBERNETES_VERSION \
        --addons=metrics-server \
        --disable-optimizations \
        --driver=docker
    
    # Verify Minikube is running
    minikube status
    
    echo "Minikube started successfully at \$(date)"
EOF
then
    echo "*** Error: Minikube startup failed"
    echo "*** Cleaning up failed instance: $INSTANCE_ID"
    aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "$INSTANCE_ID" || true
    exit 1
fi

echo "*** EC2 instance with Minikube is ready!"
echo "Instance ID: $INSTANCE_ID"
echo "Public IP: $PUBLIC_IP"

# Export environment variables for the main test script
export MINIKUBE_INSTANCE_ID="$INSTANCE_ID"
export MINIKUBE_HOST="$PUBLIC_IP"
export MINIKUBE_USER="ec2-user"
export MINIKUBE_ROOTDIR="/home/ec2-user"

# Save instance info to file for cleanup script
cat > /tmp/minikube-instance-info << EOF
MINIKUBE_INSTANCE_ID=$INSTANCE_ID
MINIKUBE_HOST=$PUBLIC_IP
MINIKUBE_USER=ec2-user
MINIKUBE_ROOTDIR=/home/ec2-user
AWS_REGION=$AWS_REGION
EOF

echo "*** Instance information saved to /tmp/minikube-instance-info"
echo "*** Provisioning completed successfully!"