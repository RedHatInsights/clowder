#!/bin/bash

set -euo pipefail

# EC2 Minikube Instance Provisioning Script
# This script provisions a new EC2 instance from a pre-built AMI with Minikube for E2E testing

# Required environment variables
: ${AWS_REGION:="us-east-1"}
: ${EC2_INSTANCE_TYPE:="m5.2xlarge"}  # 8 vCPUs, 32 GB RAM
: ${EC2_AMI_ID:?"EC2_AMI_ID must be set (your pre-built AMI with Minikube)"}
: ${EC2_KEY_PAIR_NAME:?"EC2_KEY_PAIR_NAME must be set"}
: ${EC2_SECURITY_GROUP_ID:?"EC2_SECURITY_GROUP_ID must be set"}
: ${EC2_SUBNET_ID:?"EC2_SUBNET_ID must be set"}

# Optional environment variables
: ${EC2_INSTANCE_NAME:="clowder-e2e-$(date +%Y%m%d-%H%M%S)-$(git rev-parse --short HEAD)"}
: ${KUBERNETES_VERSION:="1.30"}

echo "*** Provisioning EC2 instance for Clowder E2E tests ***"
echo "Instance Name: $EC2_INSTANCE_NAME"
echo "Instance Type: $EC2_INSTANCE_TYPE"
echo "AMI ID: $EC2_AMI_ID"
echo "Region: $AWS_REGION"

# Create minimal user data script for any additional setup if needed
cat > /tmp/ec2-userdata.sh << 'EOF'
#!/bin/bash
set -euo pipefail

# Log startup
echo "EC2 instance started at $(date)" >> /tmp/ec2-startup.log

# Ensure docker is running (should already be running from AMI)
sudo systemctl start docker || true
sudo systemctl enable docker || true

# Signal that initialization is complete
touch /tmp/ec2-init-complete

echo "EC2 instance initialization completed at $(date)" >> /tmp/ec2-startup.log
EOF

# Launch EC2 instance
echo "*** Launching EC2 instance from AMI..."
INSTANCE_ID=$(aws ec2 run-instances \
    --region "$AWS_REGION" \
    --image-id "$EC2_AMI_ID" \
    --instance-type "$EC2_INSTANCE_TYPE" \
    --key-name "$EC2_KEY_PAIR_NAME" \
    --security-group-ids "$EC2_SECURITY_GROUP_ID" \
    --subnet-id "$EC2_SUBNET_ID" \
    --user-data file:///tmp/ec2-userdata.sh \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=$EC2_INSTANCE_NAME},{Key=Purpose,Value=clowder-e2e-testing},{Key=AutoDelete,Value=true}]" \
    --query 'Instances[0].InstanceId' \
    --output text)

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

# Wait for instance initialization to complete (if needed)
echo "*** Waiting for instance initialization to complete..."
INIT_COMPLETE=false
for i in {1..30}; do
    if ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" "test -f /tmp/ec2-init-complete" 2>/dev/null; then
        echo "*** Instance initialization completed"
        INIT_COMPLETE=true
        break
    fi
    echo "Attempt $i/30: Initialization not complete, waiting 10 seconds..."
    sleep 10
done

if [ "$INIT_COMPLETE" != "true" ]; then
    echo "*** Warning: Instance initialization check timed out, but continuing..."
fi

# Start/restart Minikube (since it's pre-installed in the AMI)
echo "*** Starting Minikube on EC2 instance..."
if ! ssh -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" << EOF
    set -euo pipefail
    
    # Delete any existing minikube cluster to start fresh
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