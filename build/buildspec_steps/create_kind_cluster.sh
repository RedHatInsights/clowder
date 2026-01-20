echo "Creating kind cluster ${CLUSTER_NAME} using ${KINDEST_NODE_IMAGE}..."

# Generate Kind cluster configuration
# - Enables ValidatingAdmissionWebhook for Clowder webhooks
# - Exposes ports 80/443 for ingress testing
# - Configures containerd to use local registry
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
# Configure containerd for local image registry (if needed)
containerdConfigPatches:
- |-
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