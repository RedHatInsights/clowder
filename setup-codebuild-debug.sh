#!/usr/bin/env bash
# Script to setup Kind cluster and run tests in CodeBuild debug session
# Replicates buildspec.yml phases: install, pre_build, and build

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Variables (same as buildspec.yml)
CLUSTER_NAME=clowder-e2e
K8S_VERSION=v1.29.4
KINDEST_NODE_IMAGE=kindest/node:v1.29.4
KUTTL_VERSION="0.19.0"

echo "=========================================="
echo "CodeBuild Debug Setup Script"
echo "=========================================="
echo "This script replicates buildspec.yml phases"
echo ""

# PHASE: install (lines 13-36 of buildspec.yml)
log_info "PHASE: install - Installing tools..."

set -e
unset PYENV_VERSION
rm -f .python-version

log_info "Installing Go 1.24+ (required for go.mod)..."
curl -fsSL https://go.dev/dl/go1.24.7.linux-amd64.tar.gz | tar -C /usr/local -xzf -
export PATH="/usr/local/go/bin:$PATH"
go version

log_info "Installing base tools (jq, python3, git, make, tar, unzip)..."
yum -y install jq python3 python3-pip git make tar unzip python3-pyyaml >/dev/null 2>&1 || true
python3 -m pip install --upgrade pip virtualenv >/dev/null 2>&1 || true
export VIRTUAL_ENV=skip

log_info "Installing kind, kubectl, helm, and kuttl..."
curl -fsSL -o /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64
chmod +x /usr/local/bin/kind
curl -fsSL -o /usr/local/bin/kubectl https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl
chmod +x /usr/local/bin/kubectl
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
curl -fsSL https://github.com/kudobuilder/kuttl/releases/download/v${KUTTL_VERSION}/kubectl-kuttl_${KUTTL_VERSION}_linux_x86_64 -o /usr/local/bin/kubectl-kuttl
chmod +x /usr/local/bin/kubectl-kuttl
kubectl-kuttl version

log_info "PHASE install completed ✓"
echo ""

# PHASE: pre_build (lines 38-72 of buildspec.yml)
log_info "PHASE: pre_build - Creating Kind cluster..."

cat > kind.yaml <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
    - |
      kind: ClusterConfiguration
      apiServer:
        extraArgs:
          enable-admission-plugins: ValidatingAdmissionWebhook,MutatingAdmissionWebhook
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
EOF

kind create cluster --name "${CLUSTER_NAME}" --image "${KINDEST_NODE_IMAGE}" --config kind.yaml --wait 180s
kubectl cluster-info
kubectl get nodes -o wide

log_info "Installing ingress-nginx..."
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace
kubectl -n ingress-nginx rollout status deploy/ingress-nginx-controller --timeout=300s

log_info "PHASE pre_build completed ✓"
echo ""

# PHASE: build (lines 74-107 of buildspec.yml)
log_info "PHASE: build - Setting up operators and deploying Clowder..."

log_info "Preparing cluster dependencies (operators, CRDs)..."
export KUBECTL_CMD="kubectl"
export PATH="$PWD/bin:$PATH"
bash build/codebuild_kube_setup.sh

log_info "Building clowder manifest..."
export IMAGE_TAG=`git rev-parse --short=8 HEAD`
export IMG="quay.io/cloudservices/clowder:$IMAGE_TAG"
make release

log_info "Deploying clowder operator and config..."
kubectl create namespace clowder-system || true
kubectl apply -f manifest.yaml --validate=false -n clowder-system
kubectl apply -f clowder-config.yaml -n clowder-system
kubectl delete pod -n clowder-system -l operator-name=clowder || true

log_info "Waiting for clowder deployment..."
sleep 10
kubectl get pods -n clowder-system -o wide
kubectl rollout status deployment/clowder-controller-manager -n clowder-system --timeout=600s

log_info "PHASE build completed ✓"
echo ""

echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "Cluster is ready. To run tests:"
echo ""
echo "  # Run ALL tests:"
echo "  bash build/run_kuttl.sh"
echo ""
echo "  # Run ONE specific test:"
echo "  kubectl-kuttl test --config tests/kuttl/kuttl-test.yaml --manifest-dir config/crd/bases/ --test test-sidecars tests/kuttl/"
echo ""
echo "  # Run MULTIPLE specific tests:"
echo "  kubectl-kuttl test --config tests/kuttl/kuttl-test.yaml --manifest-dir config/crd/bases/ --test test-sidecars --test test-kafka-managed tests/kuttl/"
echo ""
echo "To cleanup:"
echo "  kind delete cluster --name ${CLUSTER_NAME}"
echo ""

