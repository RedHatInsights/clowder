#!/bin/bash

set -exv

# Dynamic EC2 Minikube E2E Testing Script (Pre-built AMI)
# This script provisions a new EC2 instance from a pre-built AMI, runs E2E tests, and cleans up

# Trap to ensure cleanup happens even on script failure
cleanup_on_exit() {
    local exit_code=$?
    echo "*** Script exiting with code: $exit_code"
    
    if [ -n "${MINIKUBE_INSTANCE_ID:-}" ]; then
        echo "*** Running cleanup due to script exit..."
        # Run cleanup in background to avoid hanging the script
        timeout 300 ./ci/cleanup_ec2_minikube.sh || echo "*** Warning: Cleanup may have timed out"
    fi
    
    exit $exit_code
}

trap cleanup_on_exit EXIT INT TERM

# Validate required AWS environment variables
: ${AWS_REGION:="us-east-1"}
: ${MINIKUBE_EC2_AMI_ID:?"MINIKUBE_EC2_AMI_ID must be set (pre-built AMI with Minikube)"}
: ${EC2_KEY_PAIR_NAME:?"EC2_KEY_PAIR_NAME must be set"}
: ${EC2_SECURITY_GROUP_ID:?"EC2_SECURITY_GROUP_ID must be set"}
: ${EC2_SUBNET_ID:?"EC2_SUBNET_ID must be set"}

# Handle private key - either as file path or content
if [ -n "${EC2_PRIVATE_KEY_CONTENT:-}" ]; then
    # For CI/CD environments, create key file from content
    export EC2_PRIVATE_KEY_PATH="/tmp/ec2-private-key.pem"
    echo -e "$EC2_PRIVATE_KEY_CONTENT" > "$EC2_PRIVATE_KEY_PATH"
    chmod 600 "$EC2_PRIVATE_KEY_PATH"
    echo "*** Created private key file from EC2_PRIVATE_KEY_CONTENT"
elif [ -n "${EC2_PRIVATE_KEY_PATH:-}" ]; then
    # For local environments, use existing file path
    if [ ! -f "$EC2_PRIVATE_KEY_PATH" ]; then
        echo "*** Error: Private key file not found at $EC2_PRIVATE_KEY_PATH"
        exit 1
    fi
    chmod 600 "$EC2_PRIVATE_KEY_PATH"
    echo "*** Using existing private key file at $EC2_PRIVATE_KEY_PATH"
else
    echo "*** Error: Either EC2_PRIVATE_KEY_PATH or EC2_PRIVATE_KEY_CONTENT must be set"
    exit 1
fi

echo "*** Starting Clowder E2E tests with dynamic EC2 provisioning (pre-built AMI) ***"

# Setup local environment (same as original script)
mkdir -p /var/workdir/bin
cd /var/workdir/bin

export KUBEBUILDER_ASSETS=/var/workdir/testbin/bin

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
echo "$(cat kubectl.sha256)  ./kubectl" | sha256sum --check
chmod +x kubectl

export ARTIFACT_PATH=/var/workdir/artifacts
mkdir -p $ARTIFACT_PATH

export PATH="/var/workdir/bin:$PATH"
cd /var/workdir/source

# Install krew (same as original script)
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

# Provision new EC2 instance with pre-built AMI
echo "*** Provisioning new EC2 instance from pre-built AMI..."
source ./ci/provision_ec2_minikube.sh

# Load instance information
if [ -f "/tmp/minikube-instance-info" ]; then
    source /tmp/minikube-instance-info
else
    echo "*** Error: Instance information not found"
    exit 1
fi

echo "*** Using Minikube instance: $MINIKUBE_INSTANCE_ID at $MINIKUBE_HOST"

set +x

# Create SSH key file for connecting to the instance (copy from the validated key file)
cp "$EC2_PRIVATE_KEY_PATH" minikube-ssh-ident
chmod 600 minikube-ssh-ident

# Get minikube IP (Minikube is already started by the provisioning script)
export MINIKUBE_IP=$(ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube ip")

set -x

# Copy certificates from the remote minikube instance
scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.key ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.crt ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/ca.crt ./

# Setup SSH port forwarding for kubectl access
ssh -o ExitOnForwardFailure=yes -f -N -L 127.0.0.1:8444:$MINIKUBE_IP:8443 -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST

# Create kubeconfig for accessing the remote cluster
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

# Setup Kubernetes environment (same as original script)
source build/kube_setup.sh

export IMAGE_TAG=$(git rev-parse --short=8 HEAD)

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

# Run the actual E2E tests
source build/run_kuttl.sh --report xml
KUTTL_RESULT=$?
mv kuttl-report.xml $ARTIFACT_PATH/junit-kuttl.xml

# Collect logs and metrics (same as original script)
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

echo "*** E2E tests completed with result: $KUTTL_RESULT"

# Cleanup will happen automatically via the trap
echo "*** Test script finished, cleanup will be triggered by trap"

exit $KUTTL_RESULT