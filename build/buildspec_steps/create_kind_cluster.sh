echo "Creating kind cluster ${CLUSTER_NAME} using ${KINDEST_NODE_IMAGE}..."

# Get ECR authentication token if AWS CLI is available
ECR_AUTH_TOKEN=""
ECR_REGISTRY="847936104033.dkr.ecr.us-east-1.amazonaws.com"
if command -v aws &> /dev/null; then
    echo "Getting ECR authentication token..."
    ECR_AUTH_TOKEN=$(aws ecr get-login-password --region us-east-1 2>/dev/null || echo "")
    if [ -n "$ECR_AUTH_TOKEN" ]; then
        echo "ECR authentication token obtained successfully"
        # Convert token to base64 auth format (username:password)
        ECR_AUTH_BASE64=$(echo -n "AWS:${ECR_AUTH_TOKEN}" | base64 -w0)
    else
        echo "WARNING: Failed to get ECR token. Private images may not be accessible."
    fi
else
    echo "WARNING: AWS CLI not found. Private ECR images will not be accessible."
fi

# Generate Kind cluster configuration
# - Enables ValidatingAdmissionWebhook for Clowder webhooks
# - Exposes ports 80/443 for ingress testing
# - Configures containerd to authenticate with ECR automatically
cat > kind.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: ClusterConfiguration
    apiServer:
      extraArgs:
        # Required for Clowder validating/mutating webhooks to work
        enable-admission-plugins: ValidatingAdmissionWebhook,MutatingAdmissionWebhook
  # Expose ports for ingress controller
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
# Configure containerd registry authentication
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.configs."${ECR_REGISTRY}".auth]
    auth = "${ECR_AUTH_BASE64}"
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5000"]
    endpoint = ["http://kind-registry:5000"]
EOF

# Create the cluster and wait up to 3 minutes for it to be ready
kind create cluster --name "${CLUSTER_NAME}" --image "${KINDEST_NODE_IMAGE}" --config kind.yaml --wait 180s
kubectl cluster-info
kubectl get nodes -o wide

# Install ingress-nginx controller (required by some tests)
echo "Installing ingress-nginx..."
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo update
helm install ingress-nginx ingress-nginx/ingress-nginx -n ingress-nginx --create-namespace
kubectl -n ingress-nginx rollout status deploy/ingress-nginx-controller --timeout=300s

# Verify ECR authentication is working
if [ -n "$ECR_AUTH_TOKEN" ]; then
    echo "Testing ECR authentication by pulling a test image..."
    docker pull "${ECR_REGISTRY}/clowder-pr-check/minio:RELEASE.2025-01-20T14-49-07Z" && echo "✓ ECR authentication working" || echo "⚠ ECR authentication may have issues"
else
    echo "⚠ Skipping ECR authentication test (no token available)"
fi

echo "Cluster is ready. Containerd is configured to authenticate with ECR automatically."
echo "All pods in any namespace will be able to pull from ${ECR_REGISTRY} without ImagePullSecrets."
