# Fail fast on any error
set -e

# Log PR context if triggered from GitHub Actions
if [ -n "${GITHUB_PR_NUMBER}" ]; then
    echo "=========================================="
    echo "Running E2E tests for PR #${GITHUB_PR_NUMBER}"
    echo "Commit: ${GITHUB_SHA}"
    echo "Branch: ${GITHUB_REF}"
    echo "Triggered by: ${GITHUB_ACTOR}"
    echo "Repository: ${GITHUB_REPOSITORY}"
    echo "=========================================="
fi

# Clear Python version conflicts
unset PYENV_VERSION
rm -f .python-version

# Install Go 1.25.3 (required for building Clowder)
# Must remove old Go first - older versions can't parse 'go 1.25.3' in go.mod
echo "Installing Go 1.25.3 (required for go.mod with patch version)..."
rm -rf /usr/local/go
curl -fsSL https://go.dev/dl/go1.25.3.linux-amd64.tar.gz | tar -C /usr/local -xzf -
export PATH="/usr/local/go/bin:$PATH"
go version

# Install system dependencies
echo "Installing base tools (jq, python3, git, make, tar, unzip)..."
yum -y install jq python3 python3-pip git make tar unzip python3-pyyaml >/dev/null 2>&1 || true
python3 -m pip install --upgrade pip virtualenv >/dev/null 2>&1 || true
export VIRTUAL_ENV=skip  # Skip venv for kube_setup.sh

# Install Kubernetes tooling
echo "Installing kind, kubectl, helm, and kuttl..."
curl -fsSL -o /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.23.0/kind-linux-amd64
chmod +x /usr/local/bin/kind
curl -fsSL -o /usr/local/bin/kubectl https://dl.k8s.io/release/${K8S_VERSION}/bin/linux/amd64/kubectl
chmod +x /usr/local/bin/kubectl
curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install kuttl test runner
echo "Installing kubectl-kuttl from GitHub releases..."
curl -fsSL https://github.com/kudobuilder/kuttl/releases/download/v0.19.0/kubectl-kuttl_0.19.0_linux_x86_64 -o /usr/local/bin/kubectl-kuttl
chmod +x /usr/local/bin/kubectl-kuttl
kubectl-kuttl version

# Configure AWS ECR authentication for pulling private images
echo "Configuring AWS ECR authentication..."
if command -v aws &> /dev/null; then
    echo "AWS CLI found, logging into ECR..."
    aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin 847936104033.dkr.ecr.us-east-1.amazonaws.com || {
        echo "WARNING: Failed to authenticate with ECR. Private images may not be accessible."
        echo "Ensure the CodeBuild role has ECR permissions (ecr:GetAuthorizationToken, ecr:BatchGetImage, etc.)"
    }
else
    echo "WARNING: AWS CLI not found. Skipping ECR authentication."
    echo "Install AWS CLI if you need to pull images from private ECR repositories."
fi