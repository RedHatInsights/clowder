#!/bin/bash

set -exv

# Function to cleanup resources on exit
cleanup_resources() {
    local exit_code=$?
    
    # Kill the SSH port forwarding process
    echo "Cleaning up SSH port forwarding..."
    pkill -f "ssh.*127.0.0.1:8444" || echo "No SSH port forwarding process found"
    
    # Only cleanup AWS resources if we're using AWS
    if [[ "$USE_AWS_INSTANCE" == "true" ]]; then
        echo "Cleaning up AWS resources..."
        if [[ -n "$AWS_INSTANCE_ID" ]]; then
            echo "Terminating EC2 instance: $AWS_INSTANCE_ID"
            aws ec2 terminate-instances --instance-ids "$AWS_INSTANCE_ID" --region "$AWS_REGION" || echo "Failed to terminate instance"
            
            # Wait for instance to terminate
            echo "Waiting for instance to terminate..."
            aws ec2 wait instance-terminated --instance-ids "$AWS_INSTANCE_ID" --region "$AWS_REGION" || echo "Failed to wait for termination"
        fi
        
        if [[ -n "$AWS_KEY_PAIR_NAME" ]]; then
            echo "Deleting key pair: $AWS_KEY_PAIR_NAME"
            aws ec2 delete-key-pair --key-name "$AWS_KEY_PAIR_NAME" --region "$AWS_REGION" || echo "Failed to delete key pair"
        fi
        
        if [[ -n "$AWS_SECURITY_GROUP_ID" ]]; then
            echo "Deleting security group: $AWS_SECURITY_GROUP_ID"
            aws ec2 delete-security-group --group-id "$AWS_SECURITY_GROUP_ID" --region "$AWS_REGION" || echo "Failed to delete security group"
        fi
    fi
    
    exit $exit_code
}

# Set trap to cleanup on exit
trap cleanup_resources EXIT

# Check if AWS credentials are provided for new AWS-based workflow
if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" && -n "$AWS_REGION" ]]; then
    echo "=== Using AWS EC2 instance for E2E tests ==="
    USE_AWS_INSTANCE=true
elif [[ -n "$MINIKUBE_HOST" && -n "$MINIKUBE_USER" && -n "$MINIKUBE_SSH_KEY" ]]; then
    echo "=== Using legacy minikube setup ==="
    USE_AWS_INSTANCE=false
else
    echo "Error: Either AWS credentials or legacy minikube configuration must be provided"
    echo "AWS workflow requires: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION"
    echo "Legacy workflow requires: MINIKUBE_HOST, MINIKUBE_USER, MINIKUBE_SSH_KEY"
    exit 1
fi

if [[ "$USE_AWS_INSTANCE" == "true" ]]; then
    # Set default values for AWS configuration
    AWS_INSTANCE_TYPE="${AWS_INSTANCE_TYPE:-t3.xlarge}"
    AWS_AMI_ID="${AWS_AMI_ID:-ami-0c02fb55956c7d316}"  # Amazon Linux 2023 AMI (us-east-1)
    AWS_SUBNET_ID="${AWS_SUBNET_ID:-}"  # Will use default VPC if not specified
    AWS_REGION="${AWS_REGION:-us-east-1}"

    # Generate unique names for this pipeline run
    PIPELINE_RUN_ID="${TEKTON_PIPELINE_RUN:-$(date +%s)-$$}"
    AWS_KEY_PAIR_NAME="clowder-e2e-${PIPELINE_RUN_ID}"
    AWS_SECURITY_GROUP_NAME="clowder-e2e-sg-${PIPELINE_RUN_ID}"
fi

mkdir -p /var/workdir/bin
cd /var/workdir/bin

export KUBEBUILDER_ASSETS=/var/workdir/testbin/bin

if [[ "$USE_AWS_INSTANCE" == "true" ]]; then
    # Install AWS CLI
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
    unzip -q awscliv2.zip
    ./aws/install --bin-dir /var/workdir/bin --install-dir /var/workdir/aws-cli
fi

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
echo "$(cat kubectl.sha256)  ./kubectl" | sha256sum --check
chmod +x kubectl

export ARTIFACT_PATH=/var/workdir/artifacts
mkdir -p $ARTIFACT_PATH

export PATH="/var/workdir/bin:$PATH"
cd /var/workdir/source
(
  cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)

export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"
export PATH="/bins:$PATH"

if [[ "$USE_AWS_INSTANCE" == "true" ]]; then
    echo "=== Provisioning AWS EC2 Instance for E2E Tests ==="

# Create a key pair for SSH access
echo "Creating AWS key pair: $AWS_KEY_PAIR_NAME"
aws ec2 create-key-pair --key-name "$AWS_KEY_PAIR_NAME" --region "$AWS_REGION" --query 'KeyMaterial' --output text > aws-minikube-key.pem
chmod 600 aws-minikube-key.pem

# Get default VPC ID if subnet not specified
if [[ -z "$AWS_SUBNET_ID" ]]; then
    echo "Getting default VPC..."
    AWS_VPC_ID=$(aws ec2 describe-vpcs --region "$AWS_REGION" --filters "Name=is-default,Values=true" --query 'Vpcs[0].VpcId' --output text)
    if [[ "$AWS_VPC_ID" == "None" ]]; then
        echo "Error: No default VPC found. Please specify AWS_SUBNET_ID"
        exit 1
    fi
    echo "Using default VPC: $AWS_VPC_ID"
    
    # Get a subnet from the default VPC
    AWS_SUBNET_ID=$(aws ec2 describe-subnets --region "$AWS_REGION" --filters "Name=vpc-id,Values=$AWS_VPC_ID" --query 'Subnets[0].SubnetId' --output text)
    echo "Using subnet: $AWS_SUBNET_ID"
else
    # Get VPC ID from subnet
    AWS_VPC_ID=$(aws ec2 describe-subnets --region "$AWS_REGION" --subnet-ids "$AWS_SUBNET_ID" --query 'Subnets[0].VpcId' --output text)
fi

# Create security group
echo "Creating security group: $AWS_SECURITY_GROUP_NAME"
AWS_SECURITY_GROUP_ID=$(aws ec2 create-security-group \
    --group-name "$AWS_SECURITY_GROUP_NAME" \
    --description "Security group for Clowder E2E tests" \
    --vpc-id "$AWS_VPC_ID" \
    --region "$AWS_REGION" \
    --query 'GroupId' --output text)

# Add SSH access rule
echo "Adding SSH access rule to security group"
aws ec2 authorize-security-group-ingress \
    --group-id "$AWS_SECURITY_GROUP_ID" \
    --protocol tcp \
    --port 22 \
    --cidr 0.0.0.0/0 \
    --region "$AWS_REGION"

# Add Kubernetes API access rule (for minikube)
echo "Adding Kubernetes API access rule to security group"
aws ec2 authorize-security-group-ingress \
    --group-id "$AWS_SECURITY_GROUP_ID" \
    --protocol tcp \
    --port 8443 \
    --cidr 0.0.0.0/0 \
    --region "$AWS_REGION"

# Create user data script for instance initialization
cat > user-data.sh << 'EOF'
#!/bin/bash
yum update -y
yum install -y docker git

# Start Docker
systemctl start docker
systemctl enable docker
usermod -a -G docker ec2-user

# Install minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Create .minikube directory for ec2-user
mkdir -p /home/ec2-user/.minikube
chown -R ec2-user:ec2-user /home/ec2-user/.minikube

# Signal that initialization is complete
touch /tmp/init-complete
EOF

# Launch EC2 instance
echo "Launching EC2 instance..."
AWS_INSTANCE_ID=$(aws ec2 run-instances \
    --image-id "$AWS_AMI_ID" \
    --count 1 \
    --instance-type "$AWS_INSTANCE_TYPE" \
    --key-name "$AWS_KEY_PAIR_NAME" \
    --security-group-ids "$AWS_SECURITY_GROUP_ID" \
    --subnet-id "$AWS_SUBNET_ID" \
    --user-data file://user-data.sh \
    --associate-public-ip-address \
    --region "$AWS_REGION" \
    --query 'Instances[0].InstanceId' --output text)

echo "EC2 Instance created: $AWS_INSTANCE_ID"

# Wait for instance to be running
echo "Waiting for instance to be running..."
aws ec2 wait instance-running --instance-ids "$AWS_INSTANCE_ID" --region "$AWS_REGION"

# Get instance public IP
AWS_INSTANCE_IP=$(aws ec2 describe-instances \
    --instance-ids "$AWS_INSTANCE_ID" \
    --region "$AWS_REGION" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' --output text)

echo "Instance is running at IP: $AWS_INSTANCE_IP"

# Wait for SSH to be available and initialization to complete
echo "Waiting for SSH access and instance initialization..."
for i in {1..60}; do
    if ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 -i aws-minikube-key.pem ec2-user@"$AWS_INSTANCE_IP" "test -f /tmp/init-complete" 2>/dev/null; then
        echo "Instance initialization complete"
        break
    fi
    echo "Waiting for initialization... (attempt $i/60)"
    sleep 30
done

if [[ $i -eq 60 ]]; then
    echo "Error: Instance initialization timed out"
    exit 1
fi

    # Set variables for the rest of the script
    export MINIKUBE_HOST="$AWS_INSTANCE_IP"
    export MINIKUBE_USER="ec2-user"
    export MINIKUBE_ROOTDIR="/home/ec2-user"

    # Create SSH key file for minikube access
    cp aws-minikube-key.pem minikube-ssh-ident
    chmod 600 minikube-ssh-ident

else
    echo "=== Using Legacy Minikube Setup ==="
    
    # Create SSH key file for minikube access from environment variable
    echo "$MINIKUBE_SSH_KEY" > minikube-ssh-ident
    chmod 600 minikube-ssh-ident
fi

set +x

# Clean up any existing minikube installation
echo "Cleaning up any existing minikube installation..."
ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube delete" || echo "No existing minikube to delete"

# Start minikube with appropriate settings
if [[ "$USE_AWS_INSTANCE" == "true" ]]; then
    echo "Starting minikube on AWS EC2 instance..."
    ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube start --driver=docker --cpus 6 --disk-size 20GB --memory 14000MB --kubernetes-version=1.30 --addons=metrics-server --disable-optimizations --apiserver-ips=$AWS_INSTANCE_IP"
else
    echo "Starting minikube on legacy instance..."
    ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube start --cpus 6 --disk-size 10GB --memory 16000MB --kubernetes-version=1.30 --addons=metrics-server --disable-optimizations"
fi

# Verify minikube is running
echo "Verifying minikube status..."
ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube status"

export MINIKUBE_IP=`ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube ip"`
echo "Minikube IP: $MINIKUBE_IP"

set -x

# Copy Kubernetes certificates from the AWS instance
echo "Copying Kubernetes certificates..."
scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.key ./ || {
    echo "Error: Failed to copy client.key"
    exit 1
}
scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.crt ./ || {
    echo "Error: Failed to copy client.crt"
    exit 1
}
scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/ca.crt ./ || {
    echo "Error: Failed to copy ca.crt"
    exit 1
}

# Set up SSH port forwarding to access Kubernetes API
echo "Setting up SSH port forwarding for Kubernetes API access..."
ssh -o ExitOnForwardFailure=yes -f -N -L 127.0.0.1:8444:$MINIKUBE_IP:8443 -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST || {
    echo "Error: Failed to set up SSH port forwarding"
    exit 1
}

# Wait a moment for the tunnel to establish
sleep 5

cat > kube-config <<- EOM
apiVersion: v1
clusters:
- cluster:
    certificate-authority: $PWD/ca.crt
    server: https://127.0.0.1:8444
  name: 127-0-0-1:8444
contexts:
- context:
    cluster: 127-0-0-1:8444
    user: remote-minikube
  name: remote-minikube
users:
- name: remote-minikube
  user:
    client-certificate: $PWD/client.crt
    client-key: $PWD/client.key
current-context: remote-minikube
kind: Config
preferences: {}
EOM

export PATH="$KUBEBUILDER_ASSETS:$PATH"

export KUBECONFIG=$PWD/kube-config
export KUBECTL_CMD="kubectl "
$KUBECTL_CMD config use-context remote-minikube
$KUBECTL_CMD get pods --all-namespaces=true

source build/kube_setup.sh

export IMAGE_TAG=`git rev-parse --short=8 HEAD`

$KUBECTL_CMD create namespace clowder-system

$KUBECTL_CMD apply -f ../manifest.yaml --validate=false -n clowder-system

## The default generated config isn't quite right for our tests - so we'll create a new one and restart clowder
$KUBECTL_CMD apply -f clowder-config.yaml -n clowder-system
$KUBECTL_CMD delete pod -n clowder-system -l operator-name=clowder

# Wait for operator deployment...
$KUBECTL_CMD rollout status deployment clowder-controller-manager -n clowder-system
$KUBECTL_CMD krew install kuttl

set +e

$KUBECTL_CMD get env

source build/run_kuttl.sh --report xml
KUTTL_RESULT=$?
mv kuttl-report.xml $ARTIFACT_PATH/junit-kuttl.xml

CLOWDER_PODS=$($KUBECTL_CMD get pod -n clowder-system -o jsonpath='{.items[*].metadata.name}')
for pod in $CLOWDER_PODS; do
    $KUBECTL_CMD logs $pod -n clowder-system > $ARTIFACT_PATH/$pod.log
    $KUBECTL_CMD logs $pod -n clowder-system | ./parse-controller-logs > $ARTIFACT_PATH/$pod-parsed-controller-logs.log
done

# Grab the metrics
$KUBECTL_CMD port-forward svc/clowder-controller-manager-metrics-service-non-auth -n clowder-system 8080 &
sleep 5
curl 127.0.0.1:8080/metrics > $ARTIFACT_PATH/clowder-metrics

STRIMZI_PODS=$($KUBECTL_CMD get pod -n strimzi -o jsonpath='{.items[*].metadata.name}')
for pod in $STRIMZI_PODS; do
    $KUBECTL_CMD logs $pod -n strimzi > $ARTIFACT_PATH/$pod.log
done

set -e

echo "=== E2E Test Execution Complete ==="
echo "KUTTL Result: $KUTTL_RESULT"

# The cleanup_resources function will be called automatically via the trap
exit $KUTTL_RESULT
