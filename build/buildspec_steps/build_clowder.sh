echo "Building clowder image locally..."

# Ensure Go 1.25.3 is in PATH (required for go.mod)
export PATH="/usr/local/go/bin:$PATH"

export IMAGE_TAG=`git rev-parse --short=8 HEAD`
export IMG="clowder:${IMAGE_TAG}"
docker build -t ${IMG} .
kind load docker-image ${IMG} --name ${CLUSTER_NAME}

echo "Generating manifest with local image ${IMG}..."
make release
