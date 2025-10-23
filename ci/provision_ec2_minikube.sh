#!/bin/bash

set -euo pipefail

# EC2 Minikube Instance Provisioning Script
# This script provisions a new EC2 instance with Minikube for E2E testing

# Required environment variables
: ${AWS_REGION:="us-east-1"}
: ${EC2_INSTANCE_TYPE:="m5.2xlarge"}  # 8 vCPUs, 32 GB RAM
: ${EC2_AMI_ID:="ami-0c02fb55956c7d316"}  # Amazon Linux 2 AMI (update as needed)
: ${EC2_KEY_PAIR_NAME:?"EC2_KEY_PAIR_NAME must be set"}
: ${EC2_SECURITY_GROUP_ID:?"EC2_SECURITY_GROUP_ID must be set"}
: ${EC2_SUBNET_ID:?"EC2_SUBNET_ID must be set"}

# Optional environment variables
: ${EC2_INSTANCE_NAME:="clowder-e2e-$(date +%Y%m%d-%H%M%S)-$(git rev-parse --short HEAD)"}
: ${MINIKUBE_VERSION:="v1.34.0"}
: ${KUBERNETES_VERSION:="1.30"}

echo "*** Provisioning EC2 instance for Clowder E2E tests ***"
echo "Instance Name: $EC2_INSTANCE_NAME"
echo "Instance Type: $EC2_INSTANCE_TYPE"
echo "Region: $AWS_REGION"

# Create user data script for EC2 instance initialization
cat > /tmp/ec2-userdata.sh << 'EOF'
#!/bin/bash
set -euo pipefail

# Update system
yum update -y

# Install Docker
yum install -y docker
systemctl start docker
systemctl enable docker
usermod -aG docker ec2-user

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl
mv kubectl /usr/local/bin/

# Install Minikube
curl -LO https://storage.googleapis.com/minikube/releases/MINIKUBE_VERSION_PLACEHOLDER/minikube-linux-amd64
chmod +x minikube-linux-amd64
mv minikube-linux-amd64 /usr/local/bin/minikube

# Install conntrack (required for minikube)
yum install -y conntrack

# Create minikube user directory
mkdir -p /home/ec2-user/.minikube
chown -R ec2-user:ec2-user /home/ec2-user/.minikube

# Signal that initialization is complete
touch /tmp/ec2-init-complete

echo "EC2 instance initialization completed at $(date)"
EOF

# Replace placeholder with actual Minikube version
sed -i "s/MINIKUBE_VERSION_PLACEHOLDER/$MINIKUBE_VERSION/g" /tmp/ec2-userdata.sh

# Launch EC2 instance
echo "*** Launching EC2 instance..."
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

# Wait for EC2 initialization to complete
echo "*** Waiting for EC2 initialization to complete..."
INIT_COMPLETE=false
for i in {1..60}; do
    if ssh -o ConnectTimeout=10 -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" "test -f /tmp/ec2-init-complete" 2>/dev/null; then
        echo "*** EC2 initialization completed"
        INIT_COMPLETE=true
        break
    fi
    echo "Attempt $i/60: Initialization not complete, waiting 30 seconds..."
    sleep 30
done

if [ "$INIT_COMPLETE" != "true" ]; then
    echo "*** Error: EC2 initialization failed after 60 attempts (30 minutes)"
    echo "*** Cleaning up failed instance: $INSTANCE_ID"
    aws ec2 terminate-instances --region "$AWS_REGION" --instance-ids "$INSTANCE_ID" || true
    exit 1
fi

# Start Minikube
echo "*** Starting Minikube on EC2 instance..."
if ! ssh -o StrictHostKeyChecking=no -i "$EC2_PRIVATE_KEY_PATH" ec2-user@"$PUBLIC_IP" << EOF
    set -euo pipefail
    
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
