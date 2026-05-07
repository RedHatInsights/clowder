# Using Clowder on MacOS

## Install Minikube

``brew install minikube``

If you do not use ``brew``, you can follow [this guide](https://v1-18.docs.kubernetes.io/docs/tasks/tools/install-minikube/)


## Run minikube

### Using Docker Driver (Recommended)

```
minikube start \
    --cpus 4 \
    --disk-size 36GB \
    --memory 16000MB \
    --driver=docker \
    --addons=registry \
    --addons=ingress \
    --addons=metrics-server \
    --disable-optimizations \
    --container-runtime=containerd \
    --kubernetes-version=v1.32.6 \
    --insecure-registry "10.0.0.0/24"
```

### Using Podman Driver

If you use podman instead of docker:

```
minikube start \
    --cpus 4 \
    --disk-size 36GB \
    --memory 16000MB \
    --driver=podman \
    --addons=registry \
    --addons=ingress \
    --addons=metrics-server \
    --disable-optimizations \
    --container-runtime=containerd \
    --kubernetes-version=v1.32.6 \
    --insecure-registry "10.0.0.0/24"
```

**Note**: The network proxy approach (step below) does not work with the podman driver. See "Using Podman Driver" section below for the workaround.
    
## Configure Minikube for Local Testing

Run script to setup Minikube cluster.
```
build/kube_setup.sh
```

## Setup Network Proxy

Next start an Alpine docker container that will act as a proxy that will connect your local machine's port 5000 to the registry running in the Minikube cluster at port 5000. You'll find more on this in the Minikube documentation for running [docker on macOS](https://minikube.sigs.k8s.io/docs/handbook/registry/#enabling-insecure-registries).
```
docker run --rm -it -d --network=host alpine ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"
```

## Run

Lastly, run the make target `deploy-minikube-quick` that will build the image locally, push the image to the registry running in Minikube, and start the pod. This command also sets a tag that will need to be updated each time you make a change locally to Clowder. This will ensure that the new pod comes up with your changes. 
```
CLOWDER_BUILD_TAG=test001 make deploy-minikube-quick
```

## Using Podman Driver

If you're using the podman driver, the network proxy approach above will not work because podman and minikube maintain separate image stores. Instead, use this workflow:

### Optional: Configure Podman for Insecure Registries

While not required for the workflow below, you can configure podman to trust local registries:

```bash
mkdir -p ~/.config/containers
cat >> ~/.config/containers/registries.conf << 'EOF'

[[registry]]
location = "127.0.0.1:5000"
insecure = true
EOF
```

### Build and Load Images

```bash
# Build the image locally with podman
CLOWDER_BUILD_TAG=test001 make docker-build-no-test-quick

# Save the image to a tar file
podman save 127.0.0.1:5000/clowder:test001 -o /tmp/clowder.tar

# Load the tar file into minikube
minikube image load /tmp/clowder.tar

# Clean up the tar file
rm /tmp/clowder.tar

# Deploy (this will use the image already in minikube)
make deploy
```

### Update Image for Changes

When you make code changes, increment the tag and repeat:

```bash
CLOWDER_BUILD_TAG=test002 make docker-build-no-test-quick
podman save 127.0.0.1:5000/clowder:test002 -o /tmp/clowder.tar
minikube image load /tmp/clowder.tar
rm /tmp/clowder.tar

# Update the deployment to use the new image
kubectl set image deployment/clowder-controller-manager \
  -n clowder-system \
  manager=127.0.0.1:5000/clowder:test002
```

### Verify

You can check that the pod is running with:
```
kubectl get pods -n clowder-system
```

### Running KUTTL Tests

Ensure the kubectl-kuttl plugin is in your PATH:

```bash
# Add krew to PATH if not already done
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

# Add to your shell profile to make it permanent
echo 'export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"' >> ~/.zshrc

# Verify kuttl is available
kubectl kuttl version

# Run tests
make kuttl KUTTL_TEST="--test=test-iqe-jobs-playwright"
```
