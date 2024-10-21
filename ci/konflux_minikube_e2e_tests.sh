#!/bin/bash

set -ev

mkdir -p /container_workspace/bin
# cp /opt/app-root/src/go/bin/* /container_workspace/bin

export KUBEBUILDER_ASSETS=/container_workspace/testbin/bin

curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
echo "$(cat kubectl.sha256)  ./kubectl" | sha256sum --check
chmod +x kubectl
mv kubectl /container_workspace/bin
export PATH="/container_workspace/bin:$PATH"

(
  cd "$(mktemp -d)" &&
  OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
  ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/\(arm\)\(64\)\?.*/\1\2/' -e 's/aarch64$/arm64/')" &&
  KREW="krew-${OS}_${ARCH}" &&
  curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
  tar zxvf "${KREW}.tar.gz" &&
  ./"${KREW}" install krew
)

source build/template_check.sh
export PATH="${KREW_ROOT:-$HOME/.krew}/bin:$PATH"

export PATH="/bins:$PATH"

echo "$MINIKUBE_SSH_KEY" > minikube-ssh-ident

chmod 600 minikube-ssh-ident

ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube delete"
ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube start --cpus 6 --disk-size 10GB --memory 16000MB --kubernetes-version=1.26.3 --addons=metrics-server --disable-optimizations"

export MINIKUBE_IP=`ssh -o StrictHostKeyChecking=no $MINIKUBE_USER@$MINIKUBE_HOST -i minikube-ssh-ident "minikube ip"`

scp -o StrictHostKeyChecking=no -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.key ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/profiles/minikube/client.crt ./
scp -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST:$MINIKUBE_ROOTDIR/.minikube/ca.crt ./

ssh -o ExitOnForwardFailure=yes -f -N -L 127.0.0.1:8444:$MINIKUBE_IP:8443 -i minikube-ssh-ident $MINIKUBE_USER@$MINIKUBE_HOST

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
export PATH="/root/go/bin:$PATH"

export KUBECONFIG=$PWD/kube-config
export KUBECTL_CMD="kubectl "
$KUBECTL_CMD config use-context remote-minikube
$KUBECTL_CMD get pods --all-namespaces=true

source build/kube_setup.sh

export IMAGE_TAG=`git rev-parse --short=8 HEAD`

$KUBECTL_CMD create namespace clowder-system
# $KUBECTL_CMD create namespace hcm-eng-prod-tenant

mkdir artifacts

make release

cat manifest.yaml > artifacts/manifest.yaml

sed -i "s/clowder:latest/clowder:$IMAGE_TAG/g" manifest.yaml

$KUBECTL_CMD apply -f manifest.yaml --validate=false -n clowder-system

## The default generated config isn't quite right for our tests - so we'll create a new one and restart clowder
$KUBECTL_CMD apply -f clowder-config.yaml -n clowder-system
$KUBECTL_CMD delete pod -n clowder-system -l operator-name=clowder

# Wait for operator deployment...
$KUBECTL_CMD rollout status deployment clowder-controller-manager -n clowder-system
$KUBECTL_CMD krew install kuttl

# curl -fsSLO "https://github.com/kudobuilder/kuttl/releases/download/v0.19.0/kubectl-kuttl_0.19.0_linux_x86_64"
# chmod +x kubectl-kuttl_0.19.0_linux_x86_64
# alias kuttl="./kubectl-kuttl_0.19.0_linux_x86_64"

set +e

$KUBECTL_CMD get env

source build/run_kuttl.sh --report xml
KUTTL_RESULT=$?
mv kuttl-report.xml artifacts/junit-kuttl.xml

CLOWDER_PODS=$($KUBECTL_CMD get pod -n clowder-system -o jsonpath='{.items[*].metadata.name}')
for pod in $CLOWDER_PODS; do
    $KUBECTL_CMD logs $pod -n clowder-system > artifacts/$pod.log
    $KUBECTL_CMD logs $pod -n clowder-system | ./parse-controller-logs > artifacts/$pod-parsed-controller-logs.log
done

# Grab the metrics
$KUBECTL_CMD port-forward svc/clowder-controller-manager-metrics-service-non-auth -n clowder-system 8080 &
sleep 5
curl 127.0.0.1:8080/metrics > artifacts/clowder-metrics

STRIMZI_PODS=$($KUBECTL_CMD get pod -n strimzi -o jsonpath='{.items[*].metadata.name}')
for pod in $STRIMZI_PODS; do
    $KUBECTL_CMD logs $pod -n strimzi > artifacts/$pod.log
done
set -e

exit $KUTTL_RESULT